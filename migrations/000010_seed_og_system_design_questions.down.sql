DELETE FROM questions WHERE title IN (
    'Design Twitter Timeline',
    'Design a Distributed Web Crawler',
    'Design Google Drive / Dropbox Sync',
    'Design Flash Sale / High-Concurrency Ticketing',
    'Design Distributed Key-Value Store (Dynamo)',
    'Design Proximity Service / Google Maps',
    'Design Real-Time Top-K / Trending Topics',
    'Design Distributed Notification Service'
);

DELETE FROM quiz_decks WHERE id IN ('deck-social-feeds', 'deck-storage-systems', 'deck-geo-systems');
