package devcli

import (
	"os"
	"path/filepath"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/ids"
)

// InitProject creates a local workspace for CLI PoC use.
func InitProject(dataDir, corpusRoot, projectID, name string) (State, error) {
	if dataDir == "" {
		return State{}, apperr.New(apperr.Validation, "--data required")
	}
	if corpusRoot == "" {
		return State{}, apperr.New(apperr.Validation, "--root required")
	}
	if projectID == "" {
		projectID = "local"
	}
	if name == "" {
		name = projectID
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return State{}, apperr.Wrap(apperr.Validation, "data dir", err)
	}
	absRoot, err := filepath.Abs(corpusRoot)
	if err != nil {
		return State{}, apperr.Wrap(apperr.Validation, "corpus root", err)
	}
	ws := Workspace{DataDir: absData}
	if err := os.MkdirAll(ws.DataDir, 0o755); err != nil {
		return State{}, apperr.Wrap(apperr.Validation, "create data dir", err)
	}
	if err := os.MkdirAll(ws.ArtifactsDir(), 0o755); err != nil {
		return State{}, apperr.Wrap(apperr.Validation, "create artifacts dir", err)
	}
	st := State{
		Project: corpus.Project{
			ID:   ids.ProjectID(projectID),
			Name: name,
		},
		CorpusRoot: absRoot,
	}
	if err := ws.Save(st); err != nil {
		return State{}, err
	}
	return st, nil
}
