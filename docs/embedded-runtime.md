# Embedded memory runtime

Available in module tag `v0.1.0`; capability version `memory-exact-v1`.
The HTTP `pkg/contextkit.Client` and API v1 remain unchanged.
[ADR-0045](decisions/0045-embedded-memory-runtime.md) defines this additive boundary.

## Use

Import `github.com/fastygo/context/pkg/contextkit/runtime` with an alias:

```go
import (
    "context"
    memory "github.com/fastygo/context/pkg/contextkit/runtime"
)

func build(ctx context.Context) (memory.PackResult, error) {
    rt, err := memory.New(ctx, memory.Config{
        ProjectID: "demo",
        MaxBytes: 256 << 10,
        Sources: []memory.Source{{
            SourceID: "ticket-1", Version: "v1",
            Text: "The account is locked.",
            TrustLevel: "project", EvidenceClass: "source_text",
        }},
    })
    if err != nil {
        return memory.PackResult{}, err
    }
    return rt.ContextPack(ctx, memory.PackRequest{
        ProjectID: "demo", Query: "account",
        Focus: memory.Focus{
            ID: "routing-v1", Objective: "Find account evidence",
            RequiredTrustLevel: "project",
            Budget: memory.Budget{MaxItems: 8, MaxChars: 4096},
        },
        PolicyRefs: []string{"routing-policy-v1"},
    })
}
```

The host authenticates the project and assigns admissible source trust/classes
before constructing the runtime. Those labels are inputs, not independently
verified authority. PolicyRefs are immutable references, not permission grants.
The embedding host resolves and enforces their meaning.

## Capabilities and ownership

One project and one immutable source snapshot per instance. Each source is one
complete UTF-8 chunk. Source text is not normalized: checksums cover exact
input bytes and spans are half-open byte offsets. IDs are logical identifiers,
not filesystem paths. Source version, bytes, trust, class, and project bind the
snapshot identity. Input order does not change results.

Search supports case-sensitive exact phrase matching only. Empty or unsupported
search options fail explicitly; there is no implicit dense/sparse/morphological
fallback. Search candidates have not passed pack trust/class gates. ContextPack
uses the existing builder to retain accepted and rejected matching material.
Nonmatching sources remain available in the returned snapshot manifest.

The runtime owns copied input and returns independent slices/JSON bytes. It is
safe for concurrent calls after construction; callers must not mutate arguments
while a call reads them. No network, filesystem, database, background workers,
model calls, or durable history are used. Drop references after the request.

## Limits and identity

Hard ceilings: 128 sources and 2 MiB encoded input per snapshot or pack request.
Config can lower these ceilings, never raise them. Control lists are capped at
128 entries. Empty texts, invalid UTF-8, NUL bytes, duplicate source ids, missing
versions, and unknown trust/classes are rejected. Budgets must be positive and
bounded; instruction bytes cannot exhaust the character budget.

MaxChars follows the existing builder's byte counting and includes instruction
reservation. MaxTokensEstimate is an optional approximate evidence-only budget.
Rejected evidence and the complete source snapshot still occupy response space.
The host must cap the final serialized response, concurrent instances, and its
transport request size before and after runtime calls. This is bounded memory
work, not a claim that resident RAM equals the input-byte ceiling.

Snapshot ID hashes `context/memory-snapshot/v1 + NUL + encoding/json(snapshot)`
with snapshot ID empty and sources sorted by id. Pack and plan IDs bind the
snapshot ID and complete PackRequest under `context/memory-pack-request/v1`.
The domain version fixes this representation. The ContextPack checksum remains
ADR-0020, not JCS. A consumer may additionally hash the entire returned bundle
under its own protocol, without replacing Context's checksum semantics.

## Retention and verification

Return PackResult (including Snapshot) together with the exact PackRequest and
module/capability versions if the consumer needs reconstruction. Recreate New
from the retained snapshot's project and sources, repeat the request, and compare
snapshot ID, pack JSON, and checksum. This reconstructs deterministic evidence,
not a model judgment or historical authenticity. No snapshot-import authority is
inferred from a matching checksum.

Cancellation is checked at public boundaries and between bounded stages.
The host must still bound input and deadlines; this profile does not promise
preemption inside each core scan or allocation.

## Verification commands

```text
go test ./pkg/contextkit/... -count=1
go test ./... -count=1
go vet ./pkg/contextkit/...
```

Run concurrent tests with the race detector on a supported Go/CGO runner.
Vercel hosting and deployment-specific performance remain consumer proof gates.
