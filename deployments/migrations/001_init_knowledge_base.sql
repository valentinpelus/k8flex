-- Initialize pgvector extension for vector similarity search
CREATE EXTENSION IF NOT EXISTS vector;

-- Create alert_cases table with vector embeddings
CREATE TABLE IF NOT EXISTS alert_cases (
    id VARCHAR(36) PRIMARY KEY,
    alert_name VARCHAR(255) NOT NULL,
    severity VARCHAR(50),
    category VARCHAR(100) NOT NULL, -- Emoji/category from LLM
    summary TEXT,
    namespace VARCHAR(255),
    pod_name VARCHAR(255),
    container_name VARCHAR(255),
    analysis TEXT NOT NULL,         -- Full LLM analysis
    debug_info TEXT,                -- Debug information collected
    validated BOOLEAN DEFAULT true,  -- Whether this case was validated
    embedding vector(1536),         -- OpenAI text-embedding-3-small dimension
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_alert_cases_alert_name ON alert_cases(alert_name);
CREATE INDEX IF NOT EXISTS idx_alert_cases_category ON alert_cases(category);
CREATE INDEX IF NOT EXISTS idx_alert_cases_severity ON alert_cases(severity);
CREATE INDEX IF NOT EXISTS idx_alert_cases_validated ON alert_cases(validated);
CREATE INDEX IF NOT EXISTS idx_alert_cases_created_at ON alert_cases(created_at DESC);

-- Create HNSW index for fast vector similarity search
-- HNSW (Hierarchical Navigable Small World) is optimized for high-dimensional vectors
CREATE INDEX IF NOT EXISTS idx_alert_cases_embedding ON alert_cases 
USING hnsw (embedding vector_cosine_ops);

-- Optional: Add a function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_alert_cases_updated_at
    BEFORE UPDATE ON alert_cases
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Create a view for quick stats
CREATE OR REPLACE VIEW alert_cases_stats AS
SELECT 
    COUNT(*) as total_cases,
    COUNT(DISTINCT category) as unique_categories,
    COUNT(DISTINCT alert_name) as unique_alerts,
    MAX(created_at) as latest_case,
    AVG(CASE WHEN created_at > NOW() - INTERVAL '7 days' THEN 1 ELSE 0 END) as cases_last_week
FROM alert_cases
WHERE validated = true;

-- Insert a comment for documentation
COMMENT ON TABLE alert_cases IS 'Stores validated alert cases with vector embeddings for similarity search';
COMMENT ON COLUMN alert_cases.embedding IS 'Vector embedding (1536-dim) for semantic similarity search';
COMMENT ON INDEX idx_alert_cases_embedding IS 'HNSW index for fast cosine similarity search on embeddings';
