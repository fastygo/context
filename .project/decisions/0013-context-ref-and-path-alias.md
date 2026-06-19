# ADR-0013: ContextRef and Path Alias for Model Context

Status: Accepted  
Date: 2026-06-17  
Related: [0011](0011-merkle-manifest-and-snapshot-namespace.md), [0014](0014-storage-role-separation.md)

## Context

Cursor obfuscates or shortens paths in vector metadata to reduce tokens and avoid
leaking machine-specific paths. The Context core must pass **clear, minimal**
context to models while preserving full provenance for citations, verification,
and replay. Merkle `path_key` serves sync; models need a different surface.

## Decision

1. **`ContextRef`:** short stable alias per chunk within a snapshot, e.g.
   `c:7f3a` (4–8 chars). Used in prompts and tool arguments, not host paths.

2. **`PathAlias`:** display path with monorepo noise stripped:
   - drop workspace root, `vendor/`, ignored paths;
   - optional module prefix only when disambiguation is required.

3. **Prompt surface (example):**

   ```text
   [c:7f3a] auth/session.go:42-89
   <snippet>
   ```

4. **Provenance table** (`chunk_aliases` in metadata store):

   ```text
   snapshot_id, chunk_id, context_ref, source_id,
   span_start, span_end, path_alias, symbol_path (optional)
   ```

5. **Index payloads** (QDrant, Tantivy) store `chunk_id` + `context_ref` +
   spans; **never** absolute filesystem paths.

6. **ContextPack builder** resolves refs → artifact slices for model; trace
   stores full provenance for replay (ADR-0006).

## Consequences

### Positive

- Lower token usage; fewer path leaks in cloud multi-tenant setups.
- Stable refs within a snapshot even when repo moves on disk.

### Negative

- Mapping table per snapshot; rebuilt on re-index.
- Human readers need alias → path resolution in UI/CLI.

### Follow-ups

- Symbol path from AST chunker (tree-sitter) as optional enricher field.
- Eval tests assert packs contain refs, not absolute paths.
