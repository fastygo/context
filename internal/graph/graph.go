// Package graph reserves graph-related identifiers and extension points.
// In-core graph storage and traversal are forever-deferred (ADR-0040 / C9);
// consumers own edge projections keyed by public project/source/chunk IDs.
package graph

import "github.com/fastygo/context/internal/ids"

// NodeRef is a reserved identity stub for consumer graph projections.
type NodeRef struct {
	ID        ids.GraphNodeID
	ProjectID ids.ProjectID
	Kind      string
}

// EdgeRef names a citation/co-occurrence/dependency edge without core storage.
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
