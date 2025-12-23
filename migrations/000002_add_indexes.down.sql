-- +migrate Down
DROP INDEX IF EXISTS idx_knowledge_bases_owner_id;
DROP INDEX IF EXISTS idx_knowledge_documents_kb_id;
DROP INDEX IF EXISTS idx_knowledge_documents_status;
DROP INDEX IF EXISTS idx_knowledge_chunks_document_id;
DROP INDEX IF EXISTS idx_knowledge_chunks_chunk_index;
