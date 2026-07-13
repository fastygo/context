package devcli

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/retrieval"
)

// FocusPutResult is CLI JSON for focus-put.
type FocusPutResult struct {
	Focus    retrieval.FocusProfile `json:"focus"`
	MetaKind string                 `json:"meta_kind,omitempty"`
}

// FocusListResult is CLI JSON for focus-list.
type FocusListResult struct {
	Focuses  []retrieval.FocusProfile `json:"focuses"`
	MetaKind string                   `json:"meta_kind,omitempty"`
}

// PutFocus stores a FocusProfile in state.json and MetadataStore when configured.
func PutFocus(dataDir, projectID string, focus retrieval.FocusProfile) (FocusPutResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return FocusPutResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return FocusPutResult{}, apperr.New(apperr.Validation, "project id mismatch")
	}
	focus.ProjectID = st.Project.ID
	if focus.ID == "" {
		focus.ID = ids.FocusID("focus_default")
	}
	if focus.RequiredTrustLevel == "" {
		focus.RequiredTrustLevel = foundation.TrustProject
	}
	if focus.CitationStrictness == "" {
		focus.CitationStrictness = "strict"
	}
	if focus.ContextBudget.MaxItems == 0 {
		focus.ContextBudget.MaxItems = 8
	}
	if focus.ContextBudget.MaxChars == 0 {
		focus.ContextBudget.MaxChars = 4000
	}
	if err := focus.Validate(); err != nil {
		return FocusPutResult{}, apperr.Wrap(apperr.Validation, "focus_profile", err)
	}

	replaced := false
	for i, f := range st.Focuses {
		if f.ID == focus.ID {
			st.Focuses[i] = focus
			replaced = true
			break
		}
	}
	if !replaced {
		st.Focuses = append(st.Focuses, focus)
	}
	if err := ws.Save(st); err != nil {
		return FocusPutResult{}, err
	}

	ctx := context.Background()
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return FocusPutResult{}, err
	}
	defer handle.Close()
	if err := handle.Store.PutProject(ctx, st.Project); err != nil {
		return FocusPutResult{}, err
	}
	if err := handle.Store.PutFocus(ctx, focus); err != nil {
		return FocusPutResult{}, err
	}
	return FocusPutResult{Focus: focus, MetaKind: string(handle.Kind)}, nil
}

// GetFocus loads a FocusProfile from MetadataStore or state.json fallback.
func GetFocus(dataDir, projectID, focusID string) (retrieval.FocusProfile, string, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return retrieval.FocusProfile{}, "", err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return retrieval.FocusProfile{}, "", apperr.New(apperr.Validation, "project id mismatch")
	}
	fid := ids.FocusID(focusID)
	if fid == "" {
		return retrieval.FocusProfile{}, "", apperr.New(apperr.Validation, "focus id required")
	}

	ctx := context.Background()
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return retrieval.FocusProfile{}, "", err
	}
	defer handle.Close()
	if handle.UsesPostgres() {
		focus, err := handle.Store.GetFocus(ctx, st.Project.ID, fid)
		if err == nil {
			return focus, string(handle.Kind), nil
		}
	}
	for _, f := range st.Focuses {
		if f.ID == fid {
			return f, "state", nil
		}
	}
	// Memory metadata may have it if PutFocus ran in-process earlier in tests.
	if focus, err := handle.Store.GetFocus(ctx, st.Project.ID, fid); err == nil {
		return focus, string(handle.Kind), nil
	}
	return retrieval.FocusProfile{}, "", apperr.New(apperr.NotFound, "focus profile not found")
}

// ListFocus returns FocusProfiles from MetadataStore (postgres) or state.json.
func ListFocus(dataDir, projectID string) (FocusListResult, error) {
	ws := Workspace{DataDir: dataDir}
	st, err := ws.Load()
	if err != nil {
		return FocusListResult{}, err
	}
	if projectID != "" && ids.ProjectID(projectID) != st.Project.ID {
		return FocusListResult{}, apperr.New(apperr.Validation, "project id mismatch")
	}
	ctx := context.Background()
	handle, err := OpenMetadata(ctx)
	if err != nil {
		return FocusListResult{}, err
	}
	defer handle.Close()
	if handle.UsesPostgres() {
		list, err := handle.Store.ListFocus(ctx, st.Project.ID)
		if err != nil {
			return FocusListResult{}, err
		}
		return FocusListResult{Focuses: list, MetaKind: string(handle.Kind)}, nil
	}
	return FocusListResult{Focuses: st.Focuses, MetaKind: "state"}, nil
}

// ParseFocusJSON builds a FocusProfile from a JSON file or inline JSON string.
func ParseFocusJSON(raw string) (retrieval.FocusProfile, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return retrieval.FocusProfile{}, apperr.New(apperr.Validation, "focus json required")
	}
	body := []byte(raw)
	if !strings.HasPrefix(raw, "{") {
		b, err := os.ReadFile(raw)
		if err != nil {
			return retrieval.FocusProfile{}, apperr.Wrap(apperr.Validation, "read focus json", err)
		}
		body = b
	}
	var focus retrieval.FocusProfile
	if err := json.Unmarshal(body, &focus); err != nil {
		return retrieval.FocusProfile{}, apperr.Wrap(apperr.Validation, "decode focus json", err)
	}
	return focus, nil
}

func resolveFocus(dataDir, projectID, focusID string) (retrieval.FocusProfile, bool, error) {
	if strings.TrimSpace(focusID) == "" {
		return retrieval.FocusProfile{}, false, nil
	}
	focus, _, err := GetFocus(dataDir, projectID, focusID)
	if err != nil {
		return retrieval.FocusProfile{}, false, err
	}
	return focus, true, nil
}
