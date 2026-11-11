CREATE TABLE discovery_searches (
 id INTEGER PRIMARY KEY,
 input TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','completed','failed','interrupted')),
 error TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 finished_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_discovery_queue ON discovery_searches(status,id);
CREATE TABLE discovery_videos (
 video_id TEXT PRIMARY KEY,
 channel_id TEXT NOT NULL,
 published_at TEXT NOT NULL,
 metadata TEXT NOT NULL
);
CREATE INDEX idx_discovery_channel ON discovery_videos(channel_id,published_at,video_id);
CREATE INDEX idx_discovery_published ON discovery_videos(published_at,video_id);
CREATE TABLE discovery_results (
 search_id INTEGER NOT NULL REFERENCES discovery_searches(id),
 video_id TEXT NOT NULL REFERENCES discovery_videos(video_id),
 PRIMARY KEY(search_id,video_id)
);
