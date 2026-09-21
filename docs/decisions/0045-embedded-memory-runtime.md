# ADR-0045: Embedded memory runtime

Status: Accepted
Date: 2026-09-21

## Context and measured blocker

The public contextkit client requires BaseURL and performs HTTP calls.
context-serve requires a data directory. Neither can build a pack inside a
request-scoped, database-free serverless consumer. Inspection of both entry
points confirms the missing public composition boundary; this is not a request
for a new retrieval engine.

## Decision

Add pkg/contextkit/runtime as an opt-in public embedded adapter. It composes
existing exact retrieval, memory index, and pack builder in this module.
The parent pkg/contextkit remains an HTTP client with no internal imports.
This narrowly extends ADR-0024 and ADR-0042's consumer boundary; HTTP API v1
and its existing behavior are unchanged.

The runtime is immutable after construction and bound to one project and
content-derived snapshot. Inputs are bounded versioned UTF-8 source texts,
one complete source per chunk, with explicit trust/evidence classes. It
supports exact case-sensitive phrase retrieval only. No dense backend,
morphology, file parsing, tool execution, durable storage, or model call is
implied. Unsupported search options fail instead of silently degrading.

The embedded adapter owns admission checks and copies, not alternate retrieval
or verdict semantics. Pack results carry the source snapshot manifest and
original texts for caller retention. Snapshot and request identities use
versioned domain-separated JSON hashing; the existing ADR-0020 pack checksum
is preserved and is not claimed to be JCS.

## Consequences

Consumers import only public Go types. A host establishes source trust and
project authorization before construction; caller labels alone are not proof.
Create one runtime per request. Release references after exporting the result;
there is no Close, background worker, filesystem access, network, or recovery
after response loss. Fixed hard ceilings can be lowered for a deployment.

A new snapshot is required for changes to source bytes, version, class, or trust.
Exact retrieval is a deliberately narrow capability, not full CLI parity.

## Verification

External-package tests cover deterministic identity, actual search and packing,
source checksums/spans, rejected trust/instruction material, ownership,
concurrency, cancellation, limits, and unsupported options. A parity test
compares the public result with the existing core builder. Full offline, Lab
smoke, and golden/adversarial suites remain required. A standalone external Go
consumer demonstrates no internal imports are needed.
