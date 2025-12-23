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
