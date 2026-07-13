// Package memory implements an in-process MetadataStore for tests and local PoC.
package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/fastygo/context/internal/agentruntime"
	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/artifacts"
	"github.com/fastygo/context/internal/corpus"
	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing"
	"github.com/fastygo/context/internal/retrieval"
	"github.com/fastygo/context/internal/storage"
	"github.com/fastygo/context/internal/tools"
	"github.com/fastygo/context/internal/tracing"
)

// Store is a thread-safe in-memory metadata adapter.
type Store struct {
	mu sync.RWMutex

	projects  map[ids.ProjectID]corpus.Project
	sources   map[string]corpus.Source
	chunks    map[string]corpus.Chunk
	snapshots map[string]indexing.IndexSnapshot
	packs     map[string]retrieval.ContextPack
	runs      map[string]agentruntime.AgentRun
	toolCalls map[string]tools.ToolCall
	traces    map[string][]tracing.Event // key: projectID|runID
	artifacts map[string]artifacts.Artifact
	lineage   map[string]artifacts.ArtifactLineage
}

// New returns an empty memory metadata store.
func New() *Store {
	return &Store{
		projects:  make(map[ids.ProjectID]corpus.Project),
		sources:   make(map[string]corpus.Source),
		chunks:    make(map[string]corpus.Chunk),
		snapshots: make(map[string]indexing.IndexSnapshot),
		packs:     make(map[string]retrieval.ContextPack),
		runs:      make(map[string]agentruntime.AgentRun),
		toolCalls: make(map[string]tools.ToolCall),
		traces:    make(map[string][]tracing.Event),
		artifacts: make(map[string]artifacts.Artifact),
		lineage:   make(map[string]artifacts.ArtifactLineage),
	}
}

func key2(a, b string) string { return a + "\x00" + b }

func (s *Store) PutProject(ctx context.Context, project corpus.Project) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := project.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "project", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[project.ID] = project
	return nil
}

func (s *Store) GetProject(ctx context.Context, id ids.ProjectID) (corpus.Project, error) {
	if err := ctx.Err(); err != nil {
		return corpus.Project{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok {
		return corpus.Project{}, apperr.New(apperr.NotFound, "project not found")
	}
	return p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]corpus.Project, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]corpus.Project, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Store) PutSource(ctx context.Context, source corpus.Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := source.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "source", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[source.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.sources[key2(string(source.ProjectID), string(source.ID))] = source
	return nil
}

func (s *Store) GetSource(ctx context.Context, projectID ids.ProjectID, sourceID ids.SourceID) (corpus.Source, error) {
	if err := ctx.Err(); err != nil {
		return corpus.Source{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	src, ok := s.sources[key2(string(projectID), string(sourceID))]
	if !ok {
		return corpus.Source{}, apperr.New(apperr.NotFound, "source not found")
	}
	return src, nil
}

func (s *Store) ListSources(ctx context.Context, projectID ids.ProjectID) ([]corpus.Source, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]corpus.Source, 0)
	prefix := string(projectID) + "\x00"
	for k, src := range s.sources {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, src)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Store) PutChunk(ctx context.Context, chunk corpus.Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := chunk.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "chunk", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[chunk.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.chunks[key2(string(chunk.ProjectID), string(chunk.ID))] = chunk
	return nil
}

func (s *Store) GetChunk(ctx context.Context, projectID ids.ProjectID, chunkID ids.ChunkID) (corpus.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return corpus.Chunk{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.chunks[key2(string(projectID), string(chunkID))]
	if !ok {
		return corpus.Chunk{}, apperr.New(apperr.NotFound, "chunk not found")
	}
	return ch, nil
}

func (s *Store) ListChunks(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) ([]corpus.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]corpus.Chunk, 0)
	for _, ch := range s.chunks {
		if ch.ProjectID == projectID && ch.SnapshotID == snapshotID {
			out = append(out, ch)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Store) PutSnapshot(ctx context.Context, snapshot indexing.IndexSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := snapshot.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "snapshot", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[snapshot.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.snapshots[key2(string(snapshot.ProjectID), string(snapshot.ID))] = snapshot
	return nil
}

func (s *Store) GetSnapshot(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) (indexing.IndexSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return indexing.IndexSnapshot{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.snapshots[key2(string(projectID), string(snapshotID))]
	if !ok {
		return indexing.IndexSnapshot{}, apperr.New(apperr.NotFound, "snapshot not found")
	}
	return snap, nil
}

func (s *Store) SetActiveSnapshot(ctx context.Context, projectID ids.ProjectID, snapshotID ids.SnapshotID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	project, ok := s.projects[projectID]
	if !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	snap, ok := s.snapshots[key2(string(projectID), string(snapshotID))]
	if !ok {
		return apperr.New(apperr.NotFound, "snapshot not found")
	}
	if snap.Status != foundation.SnapshotReady {
		return apperr.New(apperr.Conflict, "active snapshot must be ready")
	}
	if project.ActiveSnapshotID != "" && project.ActiveSnapshotID != snapshotID {
		prevKey := key2(string(projectID), string(project.ActiveSnapshotID))
		if prev, ok := s.snapshots[prevKey]; ok && prev.Status == foundation.SnapshotReady {
			prev.Status = foundation.SnapshotSuperseded
			s.snapshots[prevKey] = prev
		}
	}
	project.ActiveSnapshotID = snapshotID
	s.projects[projectID] = project
	return nil
}

func (s *Store) PutPack(ctx context.Context, pack retrieval.ContextPack) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := pack.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "context_pack", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[pack.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.packs[key2(string(pack.ProjectID), string(pack.ID))] = pack
	return nil
}

func (s *Store) GetPack(ctx context.Context, projectID ids.ProjectID, packID ids.PackID) (retrieval.ContextPack, error) {
	if err := ctx.Err(); err != nil {
		return retrieval.ContextPack{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	pack, ok := s.packs[key2(string(projectID), string(packID))]
	if !ok {
		return retrieval.ContextPack{}, apperr.New(apperr.NotFound, "context pack not found")
	}
	return pack, nil
}

func (s *Store) PutRun(ctx context.Context, run agentruntime.AgentRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := run.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "agent_run", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[run.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.runs[key2(string(run.ProjectID), string(run.ID))] = run
	return nil
}

func (s *Store) GetRun(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) (agentruntime.AgentRun, error) {
	if err := ctx.Err(); err != nil {
		return agentruntime.AgentRun{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[key2(string(projectID), string(runID))]
	if !ok {
		return agentruntime.AgentRun{}, apperr.New(apperr.NotFound, "agent run not found")
	}
	return run, nil
}

func (s *Store) PutToolCall(ctx context.Context, call tools.ToolCall) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := call.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "tool_call", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[call.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.toolCalls[key2(string(call.ProjectID), string(call.ID))] = call
	return nil
}

func (s *Store) GetToolCall(ctx context.Context, projectID ids.ProjectID, callID ids.ToolCallID) (tools.ToolCall, error) {
	if err := ctx.Err(); err != nil {
		return tools.ToolCall{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	call, ok := s.toolCalls[key2(string(projectID), string(callID))]
	if !ok {
		return tools.ToolCall{}, apperr.New(apperr.NotFound, "tool call not found")
	}
	return call, nil
}

func (s *Store) AppendTrace(ctx context.Context, event tracing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "trace_event", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[event.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	k := key2(string(event.ProjectID), string(event.RunID))
	for _, existing := range s.traces[k] {
		if existing.ID == event.ID {
			return apperr.New(apperr.Conflict, "trace event id already exists")
		}
	}
	s.traces[k] = append(s.traces[k], event)
	return nil
}

func (s *Store) ListTrace(ctx context.Context, projectID ids.ProjectID, runID ids.RunID) ([]tracing.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := s.traces[key2(string(projectID), string(runID))]
	out := make([]tracing.Event, len(events))
	copy(out, events)
	return out, nil
}

func (s *Store) PutArtifactMeta(ctx context.Context, artifact artifacts.Artifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := artifact.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "artifact", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[artifact.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	s.artifacts[key2(string(artifact.ProjectID), string(artifact.ID))] = artifact
	return nil
}

func (s *Store) GetArtifactMeta(ctx context.Context, projectID ids.ProjectID, artifactID ids.ArtifactID) (artifacts.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifacts.Artifact{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	art, ok := s.artifacts[key2(string(projectID), string(artifactID))]
	if !ok {
		return artifacts.Artifact{}, apperr.New(apperr.NotFound, "artifact meta not found")
	}
	return art, nil
}

func (s *Store) ListArtifacts(ctx context.Context, projectID ids.ProjectID) ([]artifacts.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]artifacts.Artifact, 0)
	for _, art := range s.artifacts {
		if art.ProjectID == projectID {
			out = append(out, art)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Store) PutArtifactLineage(ctx context.Context, lineage artifacts.ArtifactLineage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := lineage.Validate(); err != nil {
		return apperr.Wrap(apperr.Validation, "artifact_lineage", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[lineage.ProjectID]; !ok {
		return apperr.New(apperr.NotFound, "project not found")
	}
	outputKey := key2(string(lineage.ProjectID), string(lineage.OutputArtifactID))
	if _, ok := s.artifacts[outputKey]; !ok {
		return apperr.New(apperr.NotFound, "output artifact metadata not found")
	}
	for _, inputID := range lineage.InputArtifactIDs {
		if _, ok := s.artifacts[key2(string(lineage.ProjectID), string(inputID))]; !ok {
			return apperr.New(apperr.NotFound, "input artifact metadata not found")
		}
	}
	if _, exists := s.lineage[outputKey]; exists {
		return apperr.New(apperr.Conflict, "artifact lineage already exists")
	}
	s.lineage[outputKey] = cloneLineage(lineage)
	return nil
}

func (s *Store) GetArtifactLineage(ctx context.Context, projectID ids.ProjectID, outputArtifactID ids.ArtifactID) (artifacts.ArtifactLineage, error) {
	if err := ctx.Err(); err != nil {
		return artifacts.ArtifactLineage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	lineage, ok := s.lineage[key2(string(projectID), string(outputArtifactID))]
	if !ok {
		return artifacts.ArtifactLineage{}, apperr.New(apperr.NotFound, "artifact lineage not found")
	}
	return cloneLineage(lineage), nil
}

func (s *Store) ListArtifactLineage(ctx context.Context, projectID ids.ProjectID) ([]artifacts.ArtifactLineage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]artifacts.ArtifactLineage, 0)
	for _, lineage := range s.lineage {
		if lineage.ProjectID == projectID {
			out = append(out, cloneLineage(lineage))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].OutputArtifactID < out[j].OutputArtifactID
	})
	return out, nil
}

func cloneLineage(lineage artifacts.ArtifactLineage) artifacts.ArtifactLineage {
	lineage.InputArtifactIDs = append([]ids.ArtifactID(nil), lineage.InputArtifactIDs...)
	lineage.SourceRefs = append([]corpus.SourceRef(nil), lineage.SourceRefs...)
	return lineage
}

var (
	_ storage.MetadataStore     = (*Store)(nil)
	_ storage.ArtifactMetaStore = (*Store)(nil)
)
