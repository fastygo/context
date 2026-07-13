# Observation And Event Source Adapter Roadmap

Status: deferred plugin roadmap  
Scope: reusable adapters for time-oriented source corpora such as logs,
messages, telemetry, scientific observations, accessibility input, and
human-device interactions.

## Purpose

`fastygo/context` should preserve source identity, temporal bounds, trust,
checksums, derivation lineage, retrieval filters, `ContextPack` evidence, and
trace semantics. It must not become an IoT platform, clinical model, user
profile system, or product event taxonomy.

```text
fastygo/context
  -> neutral Source / Artifact / Chunk / TemporalRange contracts
  -> retrieval, evidence, lineage, ContextPack, AgentRun trace
  -> no device, person, reaction, capability, or session ontology

observation/event adapter
  -> source-specific event envelope and schema migrations
  -> stable identity, time, trust, ordering, batching
  -> deterministic Source/Artifact/Chunk output

product/domain plugin
  -> Device, InteractionEvent, ReactionMap, CapabilityProfile, Session, ...
```

## Core Contract Surface

Adapters consume stable core contracts only after Plan Chunk 08A proves them:

- `SourceAdapter`
- `Source`
- `Artifact` (`artifact_type`, `schema_id`)
- `ArtifactLineage`
- `TemporalRange`
- `Chunk`
- `SourceRef`
- `TrustLevel`
- `RetrievalFilters`
- `IndexSnapshot`

Runtime `tracing.Event` is not part of this adapter surface. It records engine
execution, not observed domain events.

## Adapter-Owned Event Envelope

The exact schema stays outside core. Compatible adapters should be able to
provide:

```text
event_id
schema_id / schema_version
producer_id / producer_version
occurred_at
observed_at?        # when distinct from occurred_at
ingested_at
subject_ref?        # opaque product-owned ref; never a core Person entity
payload
trust_level
```

Products decide whether the subject is a user, device, case, service, room,
experiment, or something else.

## Required Adapter Capabilities

Each adapter declares:

- stable event identity;
- ordering guarantee (`total`, `partition`, `best_effort`, `none`);
- time precision and clock source;
- late/out-of-order support;
- idempotent ingest behavior;
- schema and producer versioning;
- cursor/checkpoint or stream-offset support;
- deterministic batch/window checksum;
- trust assignment;
- redaction capability;
- retention/delete compatibility;
- temporal and metadata filter support.

Unsupported capabilities must be explicit.

## Storage Shape

Start simple:

```text
event batch (NDJSON/JSON/other media type)
  -> Artifact bytes (source truth)
  -> Source registration
  -> event-window Chunk(s)
  -> temporal metadata + checksum
  -> retrieval / ContextPack
```

Derived maps, aggregates, and reports are separate
`artifact_type=structured` outputs. They carry consumer `schema_id` and generic
`ArtifactLineage` to all source/input artifacts.

## Reusable Plugin Layers Above Core

These can be shared by several products without becoming core entities:

- generic interaction-event schemas;
- device and room registries;
- capability/preferences profiles;
- reaction or behavior aggregate schemas;
- training/session schemas;
- domain validation and visualization tools.

Promotion into core requires at least two unrelated consumers and a proven
invariant that affects retrieval, provenance, verification, or replay.

## Compatibility Tests

A shared adapter testkit should prove:

1. Same event batch + same adapter versions → same checksums and chunks.
2. Duplicate event ids do not duplicate source truth.
3. Late events create deterministic new source/snapshot versions.
4. `occurred_at` and `ingested_at` are not silently conflated.
5. Time-window retrieval is explainable and project/snapshot-scoped.
6. Derived artifacts retain lineage to multiple inputs.
7. Untrusted source bytes cannot become runtime instructions or tool policy.
8. Runtime trace and source event corpora remain separate.

## Sensitive Workload Gate

Before real human observation, accessibility, health, education, or behavioral
data:

- fine-grained ACL must filter before retrieval/model/tool steps;
- encryption and retention/delete policy must be active;
- access and policy decisions must be traced;
- product consent/guardian/legal records stay in the domain layer;
- model-produced interpretations remain inference, not source truth.

## Non-Goals

- No Arduino, MQTT, Kafka, Home Assistant, medical-device, or BCI dependency in
  core.
- No `Person`, `Patient`, `ReactionProfile`, or `TherapySession` core entity.
- No requirement that `Project` equals one person.
- No implicit clinical or psychological interpretation.
- No streaming infrastructure before a real adapter and measured need.

## Recommended Order

1. Finish Plan Chunk 08A neutral lineage/temporal contracts.
2. Prove one file-backed event fixture in Chunk 12.
3. Build a small adapter testkit.
4. Add a concrete adapter only in the consuming repository.
5. Add a broker/stream adapter after file-backed idempotency and ordering are
   proven.
