INSERT INTO questions (title, difficulty, expected_topics, time_limit_minutes) VALUES
('Design TinyURL', 'Beginner', ARRAY['Hashing', 'Base62 encoding', 'Short URL mapping', 'Redirect handling'], 45),
('Design a Rate Limiter', 'Beginner', ARRAY['Token Bucket algorithm', 'Redis counter', 'HTTP 429 status', 'Sliding Window Counter'], 45),
('Design a Distributed Financial Ledger', 'Staff', ARRAY['Double-entry bookkeeping', 'Idempotency keys', 'Event Sourcing', 'Distributed Transactions', 'Saga Pattern'], 45),
('Design a Video Streaming Infrastructure', 'Staff', ARRAY['CDN Edge Caching', 'Adaptive Bitrate Streaming', 'HLS/DASH chunks', 'Tiered Storage Architecture', 'Geo-routing'], 45);
