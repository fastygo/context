-- Bootstrap only: enable pgvector. Domain/table DDL belongs in Chunk 11+.
CREATE EXTENSION IF NOT EXISTS vector;
