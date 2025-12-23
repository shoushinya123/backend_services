-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    user_id bigserial PRIMARY KEY,
    username varchar(100) UNIQUE NOT NULL,
    email varchar(200) UNIQUE NOT NULL,
    password_hash varchar(255) NOT NULL,
    create_time timestamptz DEFAULT NOW(),
    update_time timestamptz
);

CREATE TABLE IF NOT EXISTS knowledge_bases (
    knowledge_base_id bigserial PRIMARY KEY,
    name varchar(200) NOT NULL,
    description text,
    owner_id bigint NOT NULL,
    config json,
    create_time timestamptz DEFAULT NOW(),
    update_time timestamptz,
    CONSTRAINT fk_knowledge_bases_owner FOREIGN KEY (owner_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS knowledge_documents (
    document_id bigserial PRIMARY KEY,
    knowledge_base_id bigint NOT NULL,
    title varchar(200) NOT NULL,
    content text NOT NULL,
    source varchar(20) NOT NULL,
    source_url varchar(500),
    file_path varchar(500),
    metadata json,
    status varchar(20) DEFAULT 'processing',
    vector_id varchar(255),
    total_tokens integer DEFAULT 0,
    processing_mode varchar(20) DEFAULT 'fallback',
    create_time timestamptz DEFAULT NOW(),
    update_time timestamptz,
    CONSTRAINT fk_knowledge_documents_kb FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(knowledge_base_id)
);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    chunk_id bigserial PRIMARY KEY,
    document_id bigint NOT NULL,
    content text NOT NULL,
    chunk_index integer NOT NULL,
    vector_id varchar(255) NOT NULL,
    embedding json,
    metadata json,
    token_count integer DEFAULT 0,
    prev_chunk_id bigint,
    next_chunk_id bigint,
    document_total_tokens integer DEFAULT 0,
    chunk_position integer DEFAULT 0,
    related_chunk_ids json,
    create_time timestamptz DEFAULT NOW(),
    CONSTRAINT fk_knowledge_chunks_document FOREIGN KEY (document_id) REFERENCES knowledge_documents(document_id)
);

-- +migrate Down
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS knowledge_documents;
DROP TABLE IF EXISTS knowledge_bases;
DROP TABLE IF EXISTS users;
