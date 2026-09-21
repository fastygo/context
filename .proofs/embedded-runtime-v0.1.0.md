# Embedded runtime v0.1.0 verification

Date: 2026-09-21. Environment: Windows amd64, Go 1.25.5.
Scope: ADR-0045, pkg/contextkit/runtime, capability memory-exact-v1.

## Passed

- go test ./... -count=1: full default offline suite, including HTTP/Lab,
  golden and adversarial packages, existing HTTP client and embedded runtime.
- go test ./pkg/contextkit/... -count=1: final example and public tests.
- go vet ./pkg/contextkit/...: no findings.
- External temporary Go module requiring v0.1.0 with a local replace to the
  tested checkout: public imports only, New and ContextPack succeeded.
- git diff --check: no whitespace errors.

The initial default-cache run encountered a Windows cache access error.
The successful run used a fresh GOCACHE under the OS temporary directory.

## Covered behavior

Exact search, core-builder parity, source SHA-256 and byte spans, source
version/trust/class identity, source-order independence, input/output ownership,
project isolation, trust and instruction rejection, count/byte/token budgets,
unsupported capabilities, canceled contexts, concurrent calls, and fresh
snapshot reconstruction. Original UTF-8 and CRLF bytes remain unchanged.

## Not claimed

Race instrumentation was not available: Windows has no discovered GCC,
Docker daemon is unavailable, and WSL Debian cannot mount its missing VHDX.
The concurrent test passed without the race detector. Run race tests on a
supported CI runner before making stronger concurrency assurance claims.

No Vercel deployment, latency SLO, dense retrieval, full CLI parity, live provider,
or LeX conformance was tested. Optional infrastructure-dependent tests retain
their normal skip behavior in the offline suite. Runtime is not a complete
LeX verifier and Context checksums are not JCS.
