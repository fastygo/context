package scheduler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fastygo/context/internal/apperr"
	"github.com/fastygo/context/internal/ids"
)

const DirRel = "ops/schedules"

// FileStore persists schedule specs under dataDir/ops/schedules/*.json.
type FileStore struct {
	mu      sync.Mutex
	dataDir string
}

// OpenFileStore loads an empty or existing file-backed schedule store.
func OpenFileStore(dataDir string) (*FileStore, error) {
	if dataDir == "" {
		return nil, apperr.New(apperr.Validation, "data dir required")
	}
	return &FileStore{dataDir: dataDir}, nil
}

func (s *FileStore) dir() string { return filepath.Join(s.dataDir, DirRel) }

func (s *FileStore) path(id ids.ScheduleID) string {
	return filepath.Join(s.dir(), string(id)+".json")
}

func (s *FileStore) Put(ctx context.Context, spec Spec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := spec.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir(), 0o755); err != nil {
		return apperr.Wrap(apperr.Internal, "schedule dir", err)
	}
	now := time.Now().UTC()
	if spec.CreatedAt.IsZero() {
		spec.CreatedAt = now
	}
	spec.UpdatedAt = now
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode schedule", err)
	}
	tmp := s.path(spec.ID) + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.Internal, "write schedule tmp", err)
	}
	if err := os.Rename(tmp, s.path(spec.ID)); err != nil {
		return apperr.Wrap(apperr.Internal, "replace schedule", err)
	}
	return nil
}

func (s *FileStore) Get(ctx context.Context, id ids.ScheduleID) (Spec, error) {
	if err := ctx.Err(); err != nil {
		return Spec{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readUnlocked(id)
}

func (s *FileStore) readUnlocked(id ids.ScheduleID) (Spec, error) {
	raw, err := os.ReadFile(s.path(id))
	if err != nil {
		if os.IsNotExist(err) {
			return Spec{}, apperr.New(apperr.NotFound, "schedule not found")
		}
		return Spec{}, apperr.Wrap(apperr.Internal, "read schedule", err)
	}
	var spec Spec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return Spec{}, apperr.Wrap(apperr.Internal, "decode schedule", err)
	}
	return spec, nil
}

func (s *FileStore) List(ctx context.Context, projectID ids.ProjectID) ([]Spec, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ents, err := os.ReadDir(s.dir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, apperr.Wrap(apperr.Internal, "list schedules", err)
	}
	out := make([]Spec, 0)
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.dir(), e.Name()))
		if err != nil {
			return nil, apperr.Wrap(apperr.Internal, "read schedule", err)
		}
		var spec Spec
		if err := json.Unmarshal(raw, &spec); err != nil {
			return nil, apperr.Wrap(apperr.Internal, "decode schedule", err)
		}
		if projectID != "" && spec.ProjectID != projectID {
			continue
		}
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *FileStore) Delete(ctx context.Context, id ids.ScheduleID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.path(id)); err != nil {
		if os.IsNotExist(err) {
			return apperr.New(apperr.NotFound, "schedule not found")
		}
		return apperr.Wrap(apperr.Internal, "delete schedule", err)
	}
	return nil
}

func (s *FileStore) Due(ctx context.Context, now time.Time) ([]Spec, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := s.List(ctx, "")
	if err != nil {
		return nil, err
	}
	now = now.UTC()
	out := make([]Spec, 0)
	for _, spec := range all {
		if !spec.Enabled {
			continue
		}
		if spec.Kind == KindEvent {
			continue // event schedules fire via FireEvent, not Due
		}
		if spec.NextRunAt == nil {
			continue
		}
		if !spec.NextRunAt.After(now) {
			out = append(out, spec)
		}
	}
	return out, nil
}

func (s *FileStore) MarkFired(ctx context.Context, id ids.ScheduleID, at time.Time, jobID ids.JobID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	spec, err := s.readUnlocked(id)
	if err != nil {
		return err
	}
	at = at.UTC()
	spec.LastFiredAt = &at
	spec.LastJobID = jobID
	spec.UpdatedAt = at
	switch spec.Kind {
	case KindOnceAt:
		spec.Enabled = false
		spec.NextRunAt = nil
	case KindInterval:
		next := at.Add(time.Duration(spec.IntervalSeconds) * time.Second)
		spec.NextRunAt = &next
	}
	raw, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Internal, "encode schedule", err)
	}
	tmp := s.path(id) + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return apperr.Wrap(apperr.Internal, "write schedule tmp", err)
	}
	return os.Rename(tmp, s.path(id))
}

// Tick fires due time-based schedules through enq and marks them fired.
func Tick(ctx context.Context, store Store, enq Enqueuer, now time.Time) (fired []ids.ScheduleID, err error) {
	if store == nil || enq == nil {
		return nil, apperr.New(apperr.Validation, "store and enqueuer required")
	}
	due, err := store.Due(ctx, now)
	if err != nil {
		return nil, err
	}
	for _, spec := range due {
		jobID, err := enq(ctx, spec)
		if err != nil {
			return fired, err
		}
		if err := store.MarkFired(ctx, spec.ID, now, jobID); err != nil {
			return fired, err
		}
		fired = append(fired, spec.ID)
	}
	return fired, nil
}

// FireEvent enqueues enabled event schedules matching eventType for projectID.
func FireEvent(ctx context.Context, store Store, enq Enqueuer, projectID ids.ProjectID, eventType string, now time.Time) (fired []ids.ScheduleID, err error) {
	if eventType == "" {
		return nil, apperr.New(apperr.Validation, "event_type required")
	}
	all, err := store.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, spec := range all {
		if !spec.Enabled || spec.Kind != KindEvent || spec.EventType != eventType {
			continue
		}
		jobID, err := enq(ctx, spec)
		if err != nil {
			return fired, err
		}
		if err := store.MarkFired(ctx, spec.ID, now, jobID); err != nil {
			return fired, err
		}
		fired = append(fired, spec.ID)
	}
	return fired, nil
}
