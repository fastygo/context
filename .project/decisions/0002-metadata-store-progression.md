# ADR-0002: Metadata Store Progression

Status: Accepted  
Date: 2026-06-17  
Related: [0007](0007-embedded-kv-intermediate-layers.md), [0011](0011-merkle-manifest-and-snapshot-namespace.md)

## Context

The core needs durable metadata: projects, sources, manifest chain, snapshot
pointers, chunk aliases, agent runs, and context pack snapshots. Tests need
zero-dependency adapters; local desktop use needs offline-friendly storage;
multi-user cloud needs a shared relational store.

## Decision

1. **Tests and early unit work:** in-memory metadata adapter.
2. **Local single-node PoC:** SQLite acceptable for developer machines.
3. **MVP / cloud default:** PostgreSQL for manifest chain, snapshot registry,
   chunk alias tables, and agent trace tables.
4. **Local cache / offline manifest pointer:** bbolt (see ADR-0007) mirrors
   active snapshot metadata; Postgres remains authority when online.

## Consequences

### Positive

- Same domain model across environments; only the adapter changes.
- Manifest commits and snapshot flips are transactional in Postgres.

### Negative

- Two storage paths (Postgres + local bbolt cache) require sync rules on
  reconnect.

### Follow-ups

- DDL sketch for `index_snapshots`, `chunk_aliases`, `manifest_nodes` in Plan
  Chunk 04 (Merkle manifest).
