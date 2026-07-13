package contextkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to context-serve over HTTP+JSON (ADR-0024).
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	UserAgent  string
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c *Client) url(path string, query url.Values) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return "", fmt.Errorf("contextkit: BaseURL required")
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return "", err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u, err := c.url(path, query)
	if err != nil {
		return err
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	} else {
		req.Header.Set("User-Agent", "contextkit")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		var ae APIError
		if json.Unmarshal(raw, &ae) == nil && ae.Code != "" {
			return ae
		}
		return fmt.Errorf("contextkit: HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("contextkit: decode: %w", err)
	}
	return nil
}

// Health calls GET /health.
func (c *Client) Health(ctx context.Context) (HealthResponse, error) {
	var out HealthResponse
	err := c.do(ctx, http.MethodGet, "/health", nil, nil, &out)
	return out, err
}

// Status calls GET /v1/status.
func (c *Client) Status(ctx context.Context, projectID string) (StatusResponse, error) {
	q := url.Values{}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	var out StatusResponse
	err := c.do(ctx, http.MethodGet, "/v1/status", q, nil, &out)
	return out, err
}

// Search calls POST /v1/search.
func (c *Client) Search(ctx context.Context, req SearchRequest) (SearchResult, error) {
	var out SearchResult
	err := c.do(ctx, http.MethodPost, "/v1/search", nil, req, &out)
	return out, err
}

// ContextPack calls POST /v1/context-pack.
func (c *Client) ContextPack(ctx context.Context, req PackRequest) (PackResult, error) {
	var out PackResult
	err := c.do(ctx, http.MethodPost, "/v1/context-pack", nil, req, &out)
	return out, err
}

// AgentRun calls POST /v1/agent-run.
func (c *Client) AgentRun(ctx context.Context, req PackRequest) (AgentRunResult, error) {
	var out AgentRunResult
	err := c.do(ctx, http.MethodPost, "/v1/agent-run", nil, req, &out)
	return out, err
}

// Trace calls GET /v1/trace.
func (c *Client) Trace(ctx context.Context, projectID, runID string) (TraceResult, error) {
	q := url.Values{}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	q.Set("run_id", runID)
	var out TraceResult
	err := c.do(ctx, http.MethodGet, "/v1/trace", q, nil, &out)
	return out, err
}

// FocusPut calls PUT /v1/focus.
func (c *Client) FocusPut(ctx context.Context, req FocusPutRequest) (FocusPutResult, error) {
	var out FocusPutResult
	err := c.do(ctx, http.MethodPut, "/v1/focus", nil, req, &out)
	return out, err
}

// FocusGet calls GET /v1/focus.
func (c *Client) FocusGet(ctx context.Context, projectID, focusID string) (FocusGetResult, error) {
	q := url.Values{}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	q.Set("focus_id", focusID)
	var out FocusGetResult
	err := c.do(ctx, http.MethodGet, "/v1/focus", q, nil, &out)
	return out, err
}

// FocusList calls GET /v1/focuses.
func (c *Client) FocusList(ctx context.Context, projectID string) (FocusListResult, error) {
	q := url.Values{}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	var out FocusListResult
	err := c.do(ctx, http.MethodGet, "/v1/focuses", q, nil, &out)
	return out, err
}

// Eval calls POST /v1/eval.
func (c *Client) Eval(ctx context.Context) (EvalResult, error) {
	var out EvalResult
	err := c.do(ctx, http.MethodPost, "/v1/eval", nil, map[string]any{}, &out)
	return out, err
}

// EvalHistory calls GET /v1/eval/history.
func (c *Client) EvalHistory(ctx context.Context, limit int) (EvalHistoryResult, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	var out EvalHistoryResult
	err := c.do(ctx, http.MethodGet, "/v1/eval/history", q, nil, &out)
	return out, err
}

// Metrics calls GET /v1/metrics.
func (c *Client) Metrics(ctx context.Context, projectID string) (MetricsResult, error) {
	q := url.Values{}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	var out MetricsResult
	err := c.do(ctx, http.MethodGet, "/v1/metrics", q, nil, &out)
	return out, err
}

// Ingest calls POST /v1/ingest.
func (c *Client) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	var out IngestResult
	err := c.do(ctx, http.MethodPost, "/v1/ingest", nil, req, &out)
	return out, err
}
