-- +migrate Up
-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_knowledge_bases_owner_id ON knowledge_bases(owner_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_kb_id ON knowledge_documents(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_status ON knowledge_documents(status);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_document_id ON knowledge_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_chunk_index ON knowledge_chunks(chunk_index);

-- +migrate Down
DROP INDEX IF EXISTS idx_knowledge_bases_owner_id;
DROP INDEX IF EXISTS idx_knowledge_documents_kb_id;
DROP INDEX IF EXISTS idx_knowledge_documents_status;
DROP INDEX IF EXISTS idx_knowledge_chunks_document_id;
DROP INDEX IF EXISTS idx_knowledge_chunks_chunk_index;
