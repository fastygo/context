// Package localhash is a deterministic local Embedder (Chunk 16).
// It is not a semantic model; it is a measurable, offline-stable alternative
// to models/fake HashEmbed with L2-normalized SHA256-derived vectors.
package localhash

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/fastygo/context/internal/models"
)

const (
	// Kind is the config value for CONTEXT_EMBEDDER_KIND.
	Kind = "local_hash"
	// Version is the embedding_version pin for this adapter.
	// Changing Dimension for the same Version is forbidden by config validation;
	// bump Version (e.g. local-hash-v2) when the algorithm or default dim changes.
	Version = "local-hash-v1"
	// DefaultDim is the recommended dimension for local-hash-v1.
	DefaultDim = 32
)

// Embedder produces L2-normalized dense vectors from SHA256 digests.
type Embedder struct {
	Dim     int
	Version string
}

// Embed implements models.Embedder.
func (e Embedder) Embed(ctx context.Context, texts []string) ([][]float32, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	dim := e.Dim
	if dim <= 0 {
		dim = DefaultDim
	}
	ver := e.Version
	if ver == "" {
		ver = Version
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = digestEmbed(text, dim)
	}
	return out, ver, nil
}

// Dimension reports the vector size this embedder emits.
func (e Embedder) Dimension() int {
	if e.Dim <= 0 {
		return DefaultDim
	}
	return e.Dim
}

func digestEmbed(text string, dim int) []float32 {
	out := make([]float32, dim)
	seed := sha256.Sum256([]byte(text))
	// Expand digest into dim floats via successive SHA256 blocks.
	block := seed
	for i := 0; i < dim; i++ {
		if i > 0 && i%8 == 0 {
			block = sha256.Sum256(block[:])
		}
		off := (i % 8) * 4
		u := binary.BigEndian.Uint32(block[off : off+4])
		// Map to [-1, 1).
		out[i] = float32(int32(u)) / float32(1<<31)
	}
	l2Normalize(out)
	return out
}

func l2Normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return
	}
	inv := float32(1 / math.Sqrt(sum))
	for i := range v {
		v[i] *= inv
	}
}

var _ models.Embedder = Embedder{}
