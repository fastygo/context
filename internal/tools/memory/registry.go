// Package memory implements an in-process ToolRegistry.
package memory

import (
	"sort"
	"sync"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/tools"
)

// Registry is a thread-safe tool schema registry.
type Registry struct {
	mu    sync.RWMutex
	byName map[string]tools.ToolSchema
}

func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]tools.ToolSchema)}
}

func (r *Registry) Register(schema tools.ToolSchema) error {
	if err := schema.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "tool schema", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byName[schema.Name]; ok {
		return apperr.New(apperr.Conflict, "tool already registered: "+schema.Name)
	}
	r.byName[schema.Name] = schema
	return nil
}

func (r *Registry) Get(name string) (tools.ToolSchema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byName[name]
	return s, ok
}

func (r *Registry) List() []tools.ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]tools.ToolSchema, 0, len(r.byName))
	for _, s := range r.byName {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

var _ tools.Registry = (*Registry)(nil)
