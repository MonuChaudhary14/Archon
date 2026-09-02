CREATE TABLE IF NOT EXISTS user_settings (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    target_level VARCHAR(50) DEFAULT 'senior',
    years_of_experience INTEGER DEFAULT 4,
    primary_stack VARCHAR(50) DEFAULT 'Go',
    target_companies TEXT[] DEFAULT ARRAY['Google', 'Meta', 'Stripe'],
    interviewer_strictness VARCHAR(50) DEFAULT 'standard',
    feedback_style VARCHAR(50) DEFAULT 'socratic',
    enable_proactive_hints BOOLEAN DEFAULT TRUE,
    enable_voice_interview BOOLEAN DEFAULT FALSE,
    canvas_grid_type VARCHAR(20) DEFAULT 'dots',
    snap_to_grid BOOLEAN DEFAULT TRUE,
    auto_save_interval_seconds INTEGER DEFAULT 15,
    export_format VARCHAR(10) DEFAULT 'svg',
    weekly_interview_target INTEGER DEFAULT 3,
    email_notifications BOOLEAN DEFAULT TRUE,
    weekly_report_digest BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quiz_decks (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    difficulty VARCHAR(50) NOT NULL CHECK (difficulty IN ('beginner', 'intermediate', 'senior', 'staff')),
    question_count INTEGER DEFAULT 0,
    est_minutes INTEGER DEFAULT 10,
    category VARCHAR(100) NOT NULL,
    icon_name VARCHAR(50) DEFAULT 'Layers',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quiz_questions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deck_id VARCHAR(50) REFERENCES quiz_decks(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    scenario TEXT,
    topic_tag VARCHAR(100) NOT NULL,
    options JSONB NOT NULL,
    is_daily BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_quiz_questions_deck_id ON quiz_questions(deck_id);
CREATE INDEX idx_quiz_questions_is_daily ON quiz_questions(is_daily);

CREATE TABLE IF NOT EXISTS quiz_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id VARCHAR(50) REFERENCES quiz_decks(id) ON DELETE CASCADE,
    score_percent INTEGER NOT NULL,
    correct_count INTEGER NOT NULL,
    total_questions INTEGER NOT NULL,
    time_spent_seconds INTEGER DEFAULT 0,
    answers JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_quiz_attempts_user_id ON quiz_attempts(user_id);
CREATE INDEX idx_quiz_attempts_deck_id ON quiz_attempts(deck_id);

INSERT INTO quiz_decks (id, title, description, difficulty, question_count, est_minutes, category, icon_name)
VALUES 
('deck-cap', 'CAP Theorem & Distributed Consistency', 'Master CP vs AP trade-offs, PACELC theorem, and quorum consensus.', 'senior', 2, 8, 'Distributed Systems', 'Shield'),
('deck-caching', 'Distributed Caching & Eviction', 'Cache-Aside, Write-Behind, Cache Stampede, and Redis Cluster.', 'intermediate', 2, 6, 'Performance', 'Zap')
ON CONFLICT (id) DO NOTHING;

INSERT INTO quiz_questions (deck_id, question, scenario, topic_tag, options, is_daily)
VALUES
('deck-cap', 'Under PACELC theorem, what does DynamoDB prioritize during normal (non-partitioned) operations when set to strong consistency?', 'A system administrator configures read consistency in Amazon DynamoDB.', 'Distributed Consistency', '[{"id": "cap-q1-o1", "text": "Low Latency over Consistency", "is_correct": false, "explanation": "In PACELC, PA/ELC defines Tradeoffs. Strong consistency chooses Consistency over Latency (LC)."}, {"id": "cap-q1-o2", "text": "Consistency over Latency", "is_correct": true, "explanation": "When there is no partition (E), DynamoDB trades higher latency to guarantee consistent reads (C)."}]'::jsonb, TRUE),
('deck-cap', 'Why does W + R > N guarantee read consistency in quorum-based replication?', 'Quorum configuration with N=3 replicas.', 'Quorum Consensus', '[{"id": "cap-q2-o1", "text": "Read and write quorums overlap on at least one replica.", "is_correct": true, "explanation": "Because W + R > N (2 + 2 = 4 > 3), read and write quorums overlap on at least one replica containing the latest write."}, {"id": "cap-q2-o2", "text": "It eliminates network partitions automatically.", "is_correct": false, "explanation": "Quorums do not eliminate network partitions."}]'::jsonb, FALSE),
('deck-caching', 'How do you prevent a Cache Stampede (Thundering Herd) when a key expires?', 'High QPS key in Redis expires during peak traffic.', 'Caching Strategies', '[{"id": "cache-q1-o1", "text": "Use Mutex Locking / Probabilistic Early Expiration (XFetch).", "is_correct": true, "explanation": "Mutex locks ensure only one request recomputes the cache while others wait or serve stale data."}, {"id": "cache-q1-o2", "text": "Increase Redis maximum memory limit.", "is_correct": false, "explanation": "Increasing memory does not prevent simultaneous cache key miss storms."}]'::jsonb, FALSE),
('deck-caching', 'How should you design an Idempotent Payment Webhook Processing pipeline to handle duplicate events?', 'Stripe sends payment_intent.succeeded webhooks with potential duplicates under network retries.', 'Distributed Transactions', '[{"id": "opt-1", "text": "Store webhook event_id in a unique index in SQL, verify state inside a serializable transaction.", "is_correct": true, "explanation": "An atomic unique constraint on event_id guarantees duplicate deliveries are safely acknowledged with HTTP 200 without double-crediting."}, {"id": "opt-2", "text": "Check Redis cache for event_id, if not found credit wallet and set TTL.", "is_correct": false, "explanation": "Redis without transactional locks allows race conditions under high concurrent retries."}]'::jsonb, TRUE)
ON CONFLICT DO NOTHING;
