CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    difficulty VARCHAR(50) NOT NULL CHECK (difficulty IN ('Beginner', 'Intermediate', 'Senior', 'Staff')),
    expected_topics TEXT[] NOT NULL,
    time_limit_minutes INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS interviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id),
    score INTEGER,
    started_at TIMESTAMP DEFAULT NOW(),
    ended_at TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Crucial: Explicitly index foreign keys to prevent Sequential Scans
CREATE INDEX idx_interviews_user_id ON interviews(user_id);
CREATE INDEX idx_interviews_question_id ON interviews(question_id);

INSERT INTO questions (title, difficulty, expected_topics, time_limit_minutes) VALUES
('Design WhatsApp', 'Senior', ARRAY['Load Balancer', 'WebSocket', 'Database Sharding', 'Redis', 'Kafka', 'Replication'], 45),
('Design Uber', 'Senior', ARRAY['Geospatial Indexing', 'QuadTrees', 'WebSockets', 'Consistent Hashing'], 45),
('Design Netflix', 'Intermediate', ARRAY['CDN', 'Object Storage', 'Video Encoding', 'Metadata Database', 'Caching'], 45);
