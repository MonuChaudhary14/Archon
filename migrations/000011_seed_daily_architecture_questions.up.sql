INSERT INTO quiz_questions (deck_id, question, scenario, topic_tag, options, is_daily)
VALUES
(
    'deck-cap',
    'How does Raft distributed consensus prevent split-brain leader elections during network partitions?',
    'A 5-node Raft cluster experiences an asymmetric network partition dividing nodes into a minority (2 nodes) and a majority (3 nodes).',
    'Consensus & Fault Tolerance',
    '[{"id": "raft-o1", "text": "Leader election requires a strict quorum majority (N/2 + 1 = 3 votes), preventing the minority partition from electing a leader.", "is_correct": true, "explanation": "In Raft/Paxos, state transitions and elections require strict majority quorum. 2 nodes cannot obtain 3 votes, ensuring only the partition with 3 nodes can elect a valid leader and commit entries."}, {"id": "raft-o2", "text": "Nodes in the minority partition automatically self-terminate using hardware watchdogs.", "is_correct": false, "explanation": "Nodes do not self-terminate; they continue attempting elections but fail to reach quorum and cannot commit log entries."}, {"id": "raft-o3", "text": "A central coordinator decides which partition is active.", "is_correct": false, "explanation": "Raft is a decentralized consensus protocol that does not rely on a central coordinator."}, {"id": "raft-o4", "text": "Raft switches to AP mode and accepts writes on both partitions.", "is_correct": false, "explanation": "Raft is strictly a CP protocol and rejects writes in the minority partition."}]'::jsonb,
    TRUE
),
(
    'deck-caching',
    'Why is the Cache-Aside pattern prone to race conditions without distributed locks or probabilistic expiration, and how is it mitigated?',
    'Concurrent read requests encounter a cache miss while a write request updates the database and invalidates the cache.',
    'Distributed Caching',
    '[{"id": "cache-aside-o1", "text": "A concurrent read may fetch stale DB data and repopulate cache AFTER the write invalidates it. Mitigate via Write-Through or short TTL + single-flight mutex.", "is_correct": true, "explanation": "If Reader A reads stale DB data, Writer B updates DB & deletes cache, and Reader A then writes stale data into cache, stale data persists until TTL expires. Singleflight deduplication and dual-deletion mitigate this."}, {"id": "cache-aside-o2", "text": "Cache-Aside writes directly to cache first, causing permanent database corruption.", "is_correct": false, "explanation": "Cache-Aside reads from cache, falls back to database on miss, and writes to database before deleting the cache entry."}, {"id": "cache-aside-o3", "text": "Cache-Aside disables memory eviction algorithms.", "is_correct": false, "explanation": "Cache eviction (LRU/LFU) operates independently of application cache patterns."}, {"id": "cache-aside-o4", "text": "Redis automatically locks relational tables during cache queries.", "is_correct": false, "explanation": "Redis and relational databases operate as distinct, decoupled storage engines."}]'::jsonb,
    TRUE
),
(
    'deck-storage-systems',
    'In LSM-Tree based storage engines (e.g. RocksDB, Cassandra), why are Bloom Filters essential for read performance?',
    'A read request queries a point lookup for a non-existent key across multiple SSTables on disk.',
    'Storage Engine Architecture',
    '[{"id": "lsm-o1", "text": "Bloom Filters quickly identify whether a key definitely does NOT exist in an SSTable, preventing costly random disk I/O.", "is_correct": true, "explanation": "LSM trees write sequentially to immutable SSTables. Without Bloom filters, checking for a non-existent key requires scanning every SSTable level on disk. Bloom filters yield zero false negatives."}, {"id": "lsm-o2", "text": "Bloom Filters compress SSTables to reduce disk usage by 90%.", "is_correct": false, "explanation": "Bloom filters are probabilistic set-membership data structures, not data compression algorithms."}, {"id": "lsm-o3", "text": "Bloom Filters sort records in ascending order in MemTable.", "is_correct": false, "explanation": "SkipLists or Red-Black Trees are used in MemTables for sorting, not Bloom filters."}, {"id": "lsm-o4", "text": "Bloom Filters guarantee zero false positive lookups.", "is_correct": false, "explanation": "Bloom filters have zero false negatives but can have tunable false positive rates."}]'::jsonb,
    TRUE
),
(
    'deck-social-feeds',
    'How does Consistent Hashing with Virtual Nodes prevent hot-spot imbalances when adding or removing database shards?',
    'A distributed database cluster dynamically scales from 16 to 24 nodes under fluctuating holiday traffic.',
    'Distributed Sharding',
    '[{"id": "ch-o1", "text": "Virtual nodes map each physical machine to multiple hash ring positions, distributing keys uniformly and remapping only K/N keys during resharding.", "is_correct": true, "explanation": "Consistent hashing maps nodes and keys to a 360-degree ring. Virtual nodes (e.g. 256 per server) prevent non-uniform clustering and ensure rebalancing transfers a balanced 1/N slice to the new node."}, {"id": "ch-o2", "text": "Virtual nodes duplicate 100% of all data across every node in the cluster.", "is_correct": false, "explanation": "Virtual nodes are virtual tokens on the hash ring, not full cluster replicas."}, {"id": "ch-o3", "text": "Consistent hashing eliminates the need for replication.", "is_correct": false, "explanation": "Replication is still required for high availability and fault tolerance."}, {"id": "ch-o4", "text": "Virtual nodes execute dynamic cross-shard SQL JOINs.", "is_correct": false, "explanation": "Virtual nodes are a key-partitioning technique, not a distributed query planner."}]'::jsonb,
    TRUE
),
(
    'deck-storage-systems',
    'How do Vector Clocks detect concurrent conflicts in distributed leaderless storage systems (e.g. Amazon Dynamo)?',
    'Two client nodes write conflicting updates to the same key simultaneously across separate replicas without synchronization.',
    'Distributed Consistency',
    '[{"id": "vc-o1", "text": "Each replica increments its internal counter in a clock vector; if clock A dominates clock B, A is newer, otherwise a concurrent conflict is detected.", "is_correct": true, "explanation": "Vector clocks track [node_id: counter] pairs. If neither clock vector is strictly greater than the other, both writes happened concurrently and must be resolved by application logic (e.g., shopping cart merge)."}, {"id": "vc-o2", "text": "Vector clocks synchronize hardware clocks using NTP with zero millisecond drift.", "is_correct": false, "explanation": "Vector clocks are logical clocks that operate without physical NTP clock synchronization."}, {"id": "vc-o3", "text": "Vector clocks automatically discard earlier writes without client notification.", "is_correct": false, "explanation": "Last-Write-Wins (LWW) discards writes, whereas Vector Clocks preserve concurrent branches for reconciliation."}, {"id": "vc-o4", "text": "Vector clocks enforce serializable two-phase locking.", "is_correct": false, "explanation": "Vector clocks are used for optimistic eventual consistency, not pessimistic locking."}]'::jsonb,
    TRUE
),
(
    'deck-geo-systems',
    'When implementing real-time driver location tracking for ride-sharing at 1M QPS, which indexing strategy balances write throughput and spatial queries?',
    '1 Million active drivers send GPS coordinate updates every 4 seconds while riders search for nearby drivers within 3 km.',
    'Geospatial Architecture',
    '[{"id": "geo-fast-o1", "text": "Store current coordinates in Redis GEO / In-Memory Geohash buckets with short TTL, updating memory in O(1) and performing bounding box radius searches.", "is_correct": true, "explanation": "1M updates every 4s = 250,000 writes/sec. Disk-based R-Trees/PostGIS will bottleneck on I/O. In-memory Redis GEO (sorted sets indexed by 52-bit integer geohashes) delivers sub-millisecond point updates and geospatial radius queries."}, {"id": "geo-fast-o2", "text": "Write every coordinate update to relational PostgreSQL with PostGIS GIST indexes synchronously.", "is_correct": false, "explanation": "250K synchronous disk writes/sec with B-Tree/GIST index updates causes massive disk write amplification and write lock contention."}, {"id": "geo-fast-o3", "text": "Broadcast every driver coordinate to all connected mobile clients via WebSocket fanout.", "is_correct": false, "explanation": "Broadcasting 250K updates/sec to millions of users causes severe network congestion."}, {"id": "geo-fast-o4", "text": "Store GPS traces in append-only Parquet files on S3 queried via Athena.", "is_correct": false, "explanation": "S3 and Athena have multi-second query latency and cannot support real-time 50ms driver matching."}]'::jsonb,
    TRUE
),
(
    'deck-cap',
    'Why is the Transactional Outbox Pattern combined with Change Data Capture (CDC) superior to dual-writing to SQL and Kafka in application code?',
    'An order service writes order state to PostgreSQL and publishes an OrderCreated event to Apache Kafka.',
    'Distributed Event Streaming',
    '[{"id": "outbox-o1", "text": "It guarantees atomicity via local ACID transactions; CDC streams events from the WAL to Kafka, eliminating partial failure and dual-write inconsistency.", "is_correct": true, "explanation": "If the app writes to DB and crashes before Kafka publish, or if Kafka publish succeeds but DB commit rolls back, the system enters an inconsistent state. The Outbox pattern writes state and event within the same DB transaction; Debezium/CDC reads the commit log reliably."}, {"id": "outbox-o2", "text": "The Outbox pattern eliminates the need for message brokers like Kafka.", "is_correct": false, "explanation": "The Outbox pattern works in conjunction with Kafka to ensure reliable message dispatch."}, {"id": "outbox-o3", "text": "Dual-writing guarantees exactly-once delivery across network boundaries without transactions.", "is_correct": false, "explanation": "Dual-writing cannot guarantee consistency without distributed 2PC transactions."}, {"id": "outbox-o4", "text": "CDC requires locking database tables during event consumption.", "is_correct": false, "explanation": "CDC reads binary replication logs asynchronously without locking application tables."}]'::jsonb,
    TRUE
)
ON CONFLICT DO NOTHING;
