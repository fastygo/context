// Package contextkit is the first public Go consumer surface for Context.
//
// Downstream Lab/BFF code should import this package (and speak HTTP+JSON) rather
// than github.com/fastygo/context/internal/*. Types here mirror the Chunk 20
// service contract (ADR-0024) and must not pull domain ports into pkg/.
package contextkit
