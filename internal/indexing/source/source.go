// Package source defines SourceAdapter and a local filesystem discovery adapter.
package source

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/normalize"
)

// Discovered is one source candidate from an adapter scan.
type Discovered struct {
	RelativePath string
	Bytes        []byte
	MediaType    string
	SourceType   corpus.SourceType
}

// Adapter lists project sources from an external system.
type Adapter interface {
	List(ctx context.Context, projectID ids.ProjectID, root string) ([]Discovered, error)
}

// LocalFiles discovers text/markdown files under a root directory.
// IgnorePatterns is a placeholder for future ignore-file support.
type LocalFiles struct {
	IgnorePatterns []string
}

func (a LocalFiles) List(ctx context.Context, projectID ids.ProjectID, root string) ([]Discovered, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := projectID.Validate(); err != nil {
		return nil, apperr.Wrap(apperr.Validation, "project_id", err)
	}
	if root == "" {
		return nil, apperr.New(apperr.Validation, "source root: empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, "source root", err)
	}
	var out []Discovered
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".context" {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		rel = normalize.RelativePath(rel)
		if a.ignored(rel) {
			return nil
		}
		media := mediaTypeFor(rel)
		if media == "" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return apperr.Wrap(apperr.Validation, "read source", err)
		}
		out = append(out, Discovered{
			RelativePath: rel,
			Bytes:        body,
			MediaType:    media,
			SourceType:   corpus.SourceTypeFile,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a LocalFiles) ignored(rel string) bool {
	for _, pat := range a.IgnorePatterns {
		if pat == "" {
			continue
		}
		if rel == pat || strings.HasPrefix(rel, strings.TrimSuffix(pat, "*")) && strings.HasSuffix(pat, "*") {
			return true
		}
		if rel == pat {
			return true
		}
	}
	return false
}

func mediaTypeFor(rel string) string {
	lower := strings.ToLower(rel)
	switch {
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".markdown"):
		return "text/markdown"
	case strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".text"):
		return "text/plain"
	default:
		return ""
	}
}

var _ Adapter = LocalFiles{}
