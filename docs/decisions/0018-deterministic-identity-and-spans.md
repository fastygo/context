# ADR-0018: Deterministic Identity and Spans

Status: Accepted
Date: 2026-07-11
Related: [0011](0011-merkle-manifest-and-snapshot-namespace.md),
[0013](0013-context-ref-and-path-alias.md),
[0015](0015-multilingual-linguistic-contracts.md)

## Context

Chunking, Merkle manifests, citations, and replay require identical hashes and
offsets across machines and languages. Without a written policy for path keys,
normalization, and span units, golden tests and snapshot sync will diverge.

## Decision

### 1. Canonical path keys

1. `path_key` is a stable logical key, **not** an absolute filesystem path.
2. Default construction: `path_key = hex(SHA256(project_id || 0x00 || relative_path))`
   where `relative_path` uses `/` separators, no leading `./`, and no Windows
   drive letters.
3. `relative_path` is Unicode NFC, case-sensitive as stored for the source
   adapter; adapters that need case-folding record that in source metadata, not
   by mutating `path_key` silently.
4. Model-visible paths use `ContextRef` / `PathAlias` (ADR-0013); host paths never
   enter vector/sparse payloads.

### 2. Artifact bytes vs normalized text

| Layer | Rule |
|-------|------|
| Artifact store | Store **original bytes** unchanged; checksum = SHA256(original bytes) |
| BOM | If UTF-8 BOM (`EF BB BF`) is present, strip only for **normalized text** used in hashing/chunking; artifact checksum still covers original bytes |
| Newlines | Normalize to `\n` (LF) for chunk text and Merkle leaf text; do not rewrite stored artifacts |
| Unicode | NFC for normalized text used in `chunk_hash` and token `normalized`; `surface` preserves original code points for the span |
| Encoding | Sources without valid UTF-8 are rejected or stored as binary artifacts with no text chunking until a decoder adapter exists |

### 3. Span convention

1. **Canonical offsets are byte offsets** into the newline-normalized UTF-8 text
   used for chunking (after BOM strip + NFC as above).
2. `span_start` is inclusive; `span_end` is exclusive (`[start, end)`).
3. Optional `rune_start` / `rune_end` may be stored for UI highlighting; they are
   derived, not authoritative for checksums.
4. Token and attestation spans must lie within their parent chunk span.
5. Empty spans (`start == end`) are invalid for chunks; allowed only for explicit
   zero-width markers if a future chunker needs them (not in phase 1).

### 4. Checksums and Merkle inputs

**Source leaf (Level A):**

```text
source_leaf_hash = SHA256(
  "context/source-leaf/v1" || 0x00 ||
  path_key || 0x00 ||
  source_type || 0x00 ||
  SHA256(original_artifact_bytes)
)
```

**Chunk hash (Level B):**

```text
chunk_hash = SHA256(
  "context/chunk/v1" || 0x00 ||
  chunker_version || 0x00 ||
  path_key || 0x00 ||
  uint64_be(span_start) || uint64_be(span_end) || 0x00 ||
  SHA256(normalized_chunk_text_utf8)
)
```

**Chunk set hash:** Merkle (or sorted-hash tree) over `chunk_hash` values sorted
lexicographically as hex; algorithm label `chunk_set_merkle_v1` recorded on the
snapshot.

**Source merkle root:** Git-tree-style sorted-child hashing over source leaves;
algorithm label `source_merkle_v1` on the snapshot.

Exact internal-node encoding is fixed in Chunk 04 golden tests; this ADR locks
the leaf inputs and sort order so implementations cannot diverge silently.

### 5. IDs

1. `project_id`, `source_id`, `artifact_id`, `chunk_id`, `snapshot_id` are opaque
   non-empty strings; prefer ULID/UUID text forms in PoC.
2. `chunk_id` must be stable for the same `(project_id, chunk_hash)` within a
   snapshot lineage policy defined at commit time; do not derive from host path.
3. Zero values and empty IDs fail validation.

## Consequences

### Positive

- Golden tests can pin hashes before infrastructure exists.
- Clone/move of a project directory does not break `path_key` identity.

### Negative

- Byte vs display-column confusion for some UI consumers; document rune fields.
- NFC + newline policy means "raw file viewer" and "chunk text" may differ by BOM/CRLF only.

### Follow-ups

- Chunk 04 golden vectors for CRLF, BOM, NFC/NFD pairs, and multiline spans.
- Publish exact Merkle internal-node bytes in code comments once implemented.
