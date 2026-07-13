package httpserver

// APIVersion is the frozen public HTTP contract (ADR-0026).
const APIVersion = "v1"

// APIVersionHeader is set on /health and /v1/* responses.
const APIVersionHeader = "X-Context-API-Version"
