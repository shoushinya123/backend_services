-- Migration: Add fields for long text RAG support
-- Date: 2025-01-XX
-- Description: Add token_count, chunk relationships, and processing mode fields

-- Add fields to knowledge_documents table
ALTER TABLE knowledge_documents 
ADD COLUMN IF NOT EXISTS total_tokens INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS processing_mode VARCHAR(20) DEFAULT 'fallback';

-- Add index for processing_mode
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_processing_mode ON knowledge_documents(processing_mode);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_total_tokens ON knowledge_documents(total_tokens);

-- Add fields to knowledge_chunks table
ALTER TABLE knowledge_chunks
ADD COLUMN IF NOT EXISTS token_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS prev_chunk_id BIGINT,
ADD COLUMN IF NOT EXISTS next_chunk_id BIGINT,
ADD COLUMN IF NOT EXISTS document_total_tokens INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS chunk_position INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS related_chunk_ids JSON;

-- Add indexes for chunk relationships
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_prev_chunk_id ON knowledge_chunks(prev_chunk_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_next_chunk_id ON knowledge_chunks(next_chunk_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_document_id_chunk_index ON knowledge_chunks(document_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_token_count ON knowledge_chunks(token_count);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_chunk_position ON knowledge_chunks(chunk_position);

-- Add foreign key constraints for chunk relationships (optional, can be deferred)
-- Note: These are self-referencing foreign keys, so we need to be careful
-- ALTER TABLE knowledge_chunks
-- ADD CONSTRAINT fk_knowledge_chunks_prev FOREIGN KEY (prev_chunk_id) REFERENCES knowledge_chunks(chunk_id) ON DELETE SET NULL;
-- ALTER TABLE knowledge_chunks
-- ADD CONSTRAINT fk_knowledge_chunks_next FOREIGN KEY (next_chunk_id) REFERENCES knowledge_chunks(chunk_id) ON DELETE SET NULL;

