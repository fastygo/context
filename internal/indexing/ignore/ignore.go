// Package ignore provides deterministic path ignore matching for ingest
// (Chunk 17). Patterns are a gitignore-like subset: comments, directory
// prefixes, basename globs, and path.Match on the relative slash path.
package ignore

import (
	"bufio"
	"os"
	"path"
	"strings"
)

// FileName is the project-scoped ignore file under the corpus root.
const FileName = ".contextignore"

// Defaults are applied before project .contextignore / explicit patterns.
func Defaults() []string {
	return []string{
		".git/",
		".context/",
		"vendor/",
		"node_modules/",
		"dist/",
		"build/",
		"target/",
		"bin/",
	}
}

// LoadFile reads ignore patterns from path. Missing file yields nil, nil.
func LoadFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, sc.Err()
}

// Compile merges defaults, file patterns, and extra patterns (later win only
// as additional matches — all patterns are OR'd).
func Compile(filePatterns, extra []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(list []string) {
		for _, p := range list {
			p = strings.TrimSpace(p)
			if p == "" || strings.HasPrefix(p, "#") {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	add(Defaults())
	add(filePatterns)
	add(extra)
	return out
}

// Match reports whether rel (slash-separated, relative) is ignored.
func Match(rel string, patterns []string) bool {
	rel = path.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." {
		rel = ""
	}
	for _, pat := range patterns {
		if matchOne(rel, pat) {
			return true
		}
	}
	return false
}

// MatchDir reports whether a directory relative path should be skipped entirely.
func MatchDir(rel string, patterns []string) bool {
	rel = path.Clean("/" + strings.ReplaceAll(rel, "\\", "/"))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." || rel == "" {
		return false
	}
	// Directory patterns ending with / or exact dir names.
	for _, pat := range patterns {
		p := strings.TrimSpace(pat)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		dirOnly := strings.HasSuffix(p, "/")
		p = strings.TrimSuffix(p, "/")
		p = strings.ReplaceAll(p, "\\", "/")
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
		if !strings.Contains(p, "/") {
			base := path.Base(rel)
			if ok, _ := path.Match(p, base); ok {
				return true
			}
			for _, part := range strings.Split(rel, "/") {
				if ok, _ := path.Match(p, part); ok {
					return true
				}
			}
		}
		if dirOnly {
			if ok, _ := path.Match(p, rel); ok {
				return true
			}
		}
	}
	return false
}

func matchOne(rel, pat string) bool {
	pat = strings.TrimSpace(pat)
	if pat == "" || strings.HasPrefix(pat, "#") {
		return false
	}
	pat = strings.ReplaceAll(pat, "\\", "/")
	dirOnly := strings.HasSuffix(pat, "/")
	pat = strings.TrimSuffix(pat, "/")

	if rel == pat {
		return true
	}
	if strings.HasPrefix(rel, pat+"/") {
		return true
	}
	if dirOnly {
		return false
	}
	if !strings.Contains(pat, "/") {
		base := path.Base(rel)
		if ok, _ := path.Match(pat, base); ok {
			return true
		}
		for _, part := range strings.Split(rel, "/") {
			if ok, _ := path.Match(pat, part); ok {
				return true
			}
		}
	}
	ok, _ := path.Match(pat, rel)
	return ok
}
