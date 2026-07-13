-- Chunk 15: version pins on chunk metadata rows (ADR-0011 / ADR-0018).
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS morph_version text NOT NULL DEFAULT '';
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS dictionary_version text NOT NULL DEFAULT '';
