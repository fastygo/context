# Plugin Roadmaps

Deferred adapter, resource, and methodology tracks that consume neutral Context
Runtime contracts without becoming core dependencies.

| Roadmap | Boundary |
| --- | --- |
| [language-adapters.md](language-adapters.md) | Language-specific tokenization, morphology, generation, and query expansion |
| [lexicon-resources.md](lexicon-resources.md) | Dictionaries, thesauri, attestations, historical/regional/community resources |
| [observation-event-adapters.md](observation-event-adapters.md) | Logs, messages, telemetry, observations, and interaction streams |
| [grace-vivanov.md](grace-vivanov.md) | Optional contract-first engineering methodology pack |

## Promotion Rule

A plugin concept moves into `fastygo/context` only when:

1. at least two unrelated consumers need the same invariant;
2. the invariant affects retrieval, provenance, verification, policy, or
   replay;
3. adapter-owned metadata cannot preserve it safely;
4. the contract can be named without product, brand, device, clinical, or
   methodology terminology.

Core owns neutral contracts and compatibility tests. Plugins own domain schemas,
resources, integrations, and concrete providers.
