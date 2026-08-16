CREATE TABLE IF NOT EXISTS diagram_nodes (
    id VARCHAR(255) PRIMARY KEY,
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    label VARCHAR(255) NOT NULL,
    x DOUBLE PRECISION NOT NULL,
    y DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS diagram_edges (
    id VARCHAR(255) PRIMARY KEY,
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    source VARCHAR(255) NOT NULL REFERENCES diagram_nodes(id) ON DELETE CASCADE,
    target VARCHAR(255) NOT NULL REFERENCES diagram_nodes(id) ON DELETE CASCADE,
    type VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_diagram_nodes_interview_id ON diagram_nodes(interview_id);
CREATE INDEX idx_diagram_edges_interview_id ON diagram_edges(interview_id);
