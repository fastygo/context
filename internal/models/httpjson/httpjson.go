// Package httpjson provides HTTP JSON Completer and Embedder adapters (Chunk 27).
// Protocol is minimal and brand-neutral — not a vendor SDK import.
package httpjson

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/models"
)

const (
	// ProviderID is recorded on ModelCall when the server omits provider_id.
	ProviderID = "httpjson"
	// DefaultModelVersion is used when the remote response omits model_version.
	DefaultModelVersion = "httpjson-v1"
)

// Completer POSTs to {BaseURL}/v1/complete.
type Completer struct {
	BaseURL    string
	HTTPClient *http.Client
	Provider   string
	Version    string
}

type completeRequest struct {
	ProjectID    string               `json:"project_id"`
	PackID       string               `json:"pack_id"`
	PackChecksum string               `json:"pack_checksum"`
	Messages     []models.ChatMessage `json:"messages,omitempty"`
	Evidence     []evidenceDTO        `json:"evidence"`
}

type evidenceDTO struct {
	ID      string `json:"id"`
	Surface string `json:"surface"`
	Class   string `json:"class"`
}

type completeResponse struct {
	Text         string `json:"text"`
	ProviderID   string `json:"provider_id,omitempty"`
	ModelVersion string `json:"model_version,omitempty"`
}

func (c Completer) Complete(ctx context.Context, req models.CompletionRequest) (models.CompletionResult, error) {
	if err := ctx.Err(); err != nil {
		return models.CompletionResult{}, err
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return models.CompletionResult{}, apperr.New(apperr.Validation, "http completer base URL required")
	}
	ev := make([]evidenceDTO, 0, len(req.Pack.EvidenceItems))
	for _, item := range req.Pack.EvidenceItems {
		ev = append(ev, evidenceDTO{ID: item.ID, Surface: item.Surface, Class: string(item.Class)})
	}
	body := completeRequest{
		ProjectID:    string(req.ProjectID),
		PackID:       string(req.Pack.ID),
		PackChecksum: string(req.Pack.Checksum),
		Messages:     req.Messages,
		Evidence:     ev,
	}
	var out completeResponse
	if err := postJSON(ctx, c.client(), base+"/v1/complete", body, &out); err != nil {
		return models.CompletionResult{}, err
	}
	if out.Text == "" {
		return models.CompletionResult{}, apperr.New(apperr.Unavailable, "http completer returned empty text")
	}
	prov := firstNonEmpty(out.ProviderID, c.Provider, ProviderID)
	ver := firstNonEmpty(out.ModelVersion, c.Version, DefaultModelVersion)
	sum := sha256.Sum256([]byte(out.Text + string(req.Pack.Checksum)))
	hexSum := fmt.Sprintf("%x", sum)
	call := models.ModelCall{
		ID:            ids.ModelCallID("mc_" + hexSum[:16]),
		ProjectID:     req.ProjectID,
		PackID:        req.Pack.ID,
		ProviderID:    prov,
		ModelVersion:  ver,
		InputChecksum: hexSum,
		Status:        "completed",
	}
	return models.CompletionResult{Text: out.Text, ModelCall: call}, nil
}

// Embedder POSTs to {BaseURL}/v1/embed.
type Embedder struct {
	BaseURL    string
	HTTPClient *http.Client
	Version    string
	Dimension  int
}

type embedRequest struct {
	Texts []string `json:"texts"`
}

type embedResponse struct {
	Vectors      [][]float32 `json:"vectors"`
	ModelVersion string      `json:"model_version,omitempty"`
}

func (e Embedder) Embed(ctx context.Context, texts []string) ([][]float32, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	base := strings.TrimRight(e.BaseURL, "/")
	if base == "" {
		return nil, "", apperr.New(apperr.Validation, "http embedder base URL required")
	}
	var out embedResponse
	if err := postJSON(ctx, e.client(), base+"/v1/embed", embedRequest{Texts: texts}, &out); err != nil {
		return nil, "", err
	}
	if len(out.Vectors) != len(texts) {
		return nil, "", apperr.New(apperr.Unavailable, fmt.Sprintf("http embedder vector count %d != texts %d", len(out.Vectors), len(texts)))
	}
	if e.Dimension > 0 {
		for i, v := range out.Vectors {
			if len(v) != e.Dimension {
				return nil, "", apperr.New(apperr.Validation, fmt.Sprintf("http embedder dim mismatch at %d: got %d want %d", i, len(v), e.Dimension))
			}
		}
	}
	ver := firstNonEmpty(out.ModelVersion, e.Version, DefaultModelVersion)
	return out.Vectors, ver, nil
}

func (c Completer) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (e Embedder) client() *http.Client {
	if e.HTTPClient != nil {
		return e.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func postJSON(ctx context.Context, client *http.Client, url string, in any, out any) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode httpjson body", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return apperr.Wrap(apperr.Internal, "httpjson request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return apperr.Wrap(apperr.Unavailable, "httpjson transport", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return apperr.Wrap(apperr.Unavailable, "httpjson read", err)
	}
	if res.StatusCode >= 400 {
		return apperr.New(apperr.Unavailable, fmt.Sprintf("httpjson HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body))))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return apperr.Wrap(apperr.Unavailable, "httpjson decode", err)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var (
	_ models.Completer = Completer{}
	_ models.Embedder  = Embedder{}
)
