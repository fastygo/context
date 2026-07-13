// Package graph reserves graph-related identifiers and extension points.
// Graph traversal and storage are intentionally not implemented in Chunk 02.
package graph

import "github.com/fastygo/context/internal/ids"

// NodeRef is an optional retrieval filter extension point (see retrieval.RetrievalFilters).
type NodeRef struct {
	ID        ids.GraphNodeID
	ProjectID ids.ProjectID
	Kind      string
}

// EdgeRef names a future citation/co-occurrence/dependency edge without storage.
type EdgeRef struct {
	ID        ids.GraphEdgeID
	ProjectID ids.ProjectID
	From      ids.GraphNodeID
	To        ids.GraphNodeID
	Kind      string
}

func (n NodeRef) Validate() error {
	if err := n.ID.Validate(); err != nil {
		return err
	}
	return n.ProjectID.Validate()
}

func (e EdgeRef) Validate() error {
	if err := e.ID.Validate(); err != nil {
		return err
	}
	if err := e.ProjectID.Validate(); err != nil {
		return err
	}
	if err := e.From.Validate(); err != nil {
		return err
	}
	return e.To.Validate()
}
