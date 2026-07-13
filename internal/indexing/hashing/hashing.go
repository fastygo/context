// Package hashing implements ADR-0018 path keys and dual Merkle leaf/root hashes.
package hashing

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"

	"github.com/fastygo/context/internal/foundation"
	"github.com/fastygo/context/internal/ids"
	"github.com/fastygo/context/internal/indexing/normalize"
)

// PathKey returns hex(SHA256(project_id || 0x00 || relative_path)).
func PathKey(projectID ids.ProjectID, relativePath string) string {
	rel := normalize.RelativePath(relativePath)
	sum := sha256.New()
	sum.Write([]byte(projectID))
	sum.Write([]byte{0x00})
	sum.Write([]byte(rel))
	return hex.EncodeToString(sum.Sum(nil))
}

// ArtifactChecksum is SHA256 of original artifact bytes.
func ArtifactChecksum(original []byte) foundation.ChecksumHex {
	sum := sha256.Sum256(original)
	return foundation.ChecksumHex(hex.EncodeToString(sum[:]))
}

// SourceLeafHash computes Level-A leaf hash (ADR-0018).
func SourceLeafHash(pathKey, sourceType string, original []byte) foundation.ChecksumHex {
	art := ArtifactChecksum(original)
	sum := sha256.New()
	sum.Write([]byte(foundation.SourceLeafDomain))
	sum.Write([]byte{0x00})
	sum.Write([]byte(pathKey))
	sum.Write([]byte{0x00})
	sum.Write([]byte(sourceType))
	sum.Write([]byte{0x00})
	sum.Write([]byte(art))
	return foundation.ChecksumHex(hex.EncodeToString(sum.Sum(nil)))
}

// ChunkHash computes Level-B chunk hash (ADR-0018).
func ChunkHash(chunkerVersion, pathKey string, spanStart, spanEnd uint64, normalizedChunkText string) foundation.ChecksumHex {
	textSum := sha256.Sum256([]byte(normalizedChunkText))
	sum := sha256.New()
	sum.Write([]byte(foundation.ChunkHashDomain))
	sum.Write([]byte{0x00})
	sum.Write([]byte(chunkerVersion))
	sum.Write([]byte{0x00})
	sum.Write([]byte(pathKey))
	sum.Write([]byte{0x00})
	var be [16]byte
	binary.BigEndian.PutUint64(be[0:8], spanStart)
	binary.BigEndian.PutUint64(be[8:16], spanEnd)
	sum.Write(be[:])
	sum.Write([]byte{0x00})
	sum.Write(textSum[:])
	return foundation.ChecksumHex(hex.EncodeToString(sum.Sum(nil)))
}

// SourceMerkleRoot is a flat sorted-child tree over path_key → leaf_hash.
// Algorithm label: source_merkle_v1.
func SourceMerkleRoot(leaves map[string]foundation.ChecksumHex) foundation.ChecksumHex {
	keys := sortedKeys(leaves)
	sum := sha256.New()
	sum.Write([]byte(foundation.SourceMerkleAlgo))
	sum.Write([]byte{0x00})
	for _, k := range keys {
		sum.Write([]byte(k))
		sum.Write([]byte{0x00})
		sum.Write([]byte(leaves[k]))
		sum.Write([]byte{0x00})
	}
	return foundation.ChecksumHex(hex.EncodeToString(sum.Sum(nil)))
}

// ChunkSetHash is a flat sorted-child tree over chunk hashes.
// Algorithm label: chunk_set_merkle_v1.
func ChunkSetHash(chunkHashes []foundation.ChecksumHex) foundation.ChecksumHex {
	vals := make([]string, len(chunkHashes))
	for i, h := range chunkHashes {
		vals[i] = string(h)
	}
	sortStrings(vals)
	sum := sha256.New()
	sum.Write([]byte(foundation.ChunkSetMerkleAlgo))
	sum.Write([]byte{0x00})
	for _, h := range vals {
		sum.Write([]byte(h))
		sum.Write([]byte{0x00})
	}
	return foundation.ChecksumHex(hex.EncodeToString(sum.Sum(nil)))
}

func sortedKeys(m map[string]foundation.ChecksumHex) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortStrings(vals []string) {
	sort.Strings(vals)
}
