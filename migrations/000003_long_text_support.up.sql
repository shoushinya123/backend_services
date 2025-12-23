-- +migrate Up
-- Add fields for long text RAG support
ALTER TABLE knowledge_documents 
ADD COLUMN IF NOT EXISTS total_tokens INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS processing_mode VARCHAR(20) DEFAULT 'fallback';

ALTER TABLE knowledge_chunks 
ADD COLUMN IF NOT EXISTS token_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS prev_chunk_id BIGINT,
ADD COLUMN IF NOT EXISTS next_chunk_id BIGINT,
ADD COLUMN IF NOT EXISTS document_total_tokens INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS chunk_position INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS related_chunk_ids JSON;

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_processing_mode ON knowledge_documents(processing_mode);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_total_tokens ON knowledge_documents(total_tokens);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_prev_chunk_id ON knowledge_chunks(prev_chunk_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_next_chunk_id ON knowledge_chunks(next_chunk_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_token_count ON knowledge_chunks(token_count);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_chunk_position ON knowledge_chunks(chunk_position);

-- +migrate Down
DROP INDEX IF EXISTS idx_knowledge_documents_processing_mode;
DROP INDEX IF EXISTS idx_knowledge_documents_total_tokens;
DROP INDEX IF EXISTS idx_knowledge_chunks_prev_chunk_id;
DROP INDEX IF EXISTS idx_knowledge_chunks_next_chunk_id;
DROP INDEX IF EXISTS idx_knowledge_chunks_token_count;
DROP INDEX IF EXISTS idx_knowledge_chunks_chunk_position;

ALTER TABLE knowledge_documents 
DROP COLUMN IF EXISTS total_tokens,
DROP COLUMN IF EXISTS processing_mode;

ALTER TABLE knowledge_chunks 
DROP COLUMN IF EXISTS token_count,
DROP COLUMN IF EXISTS prev_chunk_id,
DROP COLUMN IF EXISTS next_chunk_id,
DROP COLUMN IF EXISTS document_total_tokens,
DROP COLUMN IF EXISTS chunk_position,
DROP COLUMN IF EXISTS related_chunk_ids;
