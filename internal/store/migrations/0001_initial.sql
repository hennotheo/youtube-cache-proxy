-- Persistent download queue and metadata. Timestamps are UTC ISO 8601 strings.
CREATE TABLE sources (
    id INTEGER PRIMARY KEY,
    video_id TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'downloading', 'ready', 'failed')),
    title TEXT,
    duration_seconds INTEGER CHECK (duration_seconds IS NULL OR duration_seconds >= 0),
    audio_path TEXT UNIQUE,
    audio_mime TEXT,
    audio_size_bytes INTEGER CHECK (audio_size_bytes IS NULL OR audio_size_bytes >= 0),
    error_code TEXT,
    error_message TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT
);

CREATE INDEX idx_sources_status_created ON sources (status, created_at, id);

-- Runs record manual selections, including batches stopped before completion.
CREATE TABLE download_runs (
    id INTEGER PRIMARY KEY,
    requested_count INTEGER NOT NULL CHECK (requested_count > 0),
    reserved_count INTEGER NOT NULL DEFAULT 0 CHECK (reserved_count >= 0 AND reserved_count <= requested_count),
    status TEXT NOT NULL CHECK (status IN ('active', 'completed', 'partial', 'interrupted')),
    stop_reason TEXT,
    created_at TEXT NOT NULL,
    completed_at TEXT
);

-- Snapshot video_id so deleting a source does not erase run history.
CREATE TABLE download_run_sources (
    id INTEGER PRIMARY KEY,
    run_id INTEGER NOT NULL REFERENCES download_runs(id) ON DELETE CASCADE,
    source_id INTEGER REFERENCES sources(id) ON DELETE SET NULL,
    video_id TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position > 0),
    UNIQUE (run_id, video_id),
    UNIQUE (run_id, position)
);

CREATE INDEX idx_run_sources_run_position ON download_run_sources (run_id, position);

-- Attempts are kept after source deletion for rate accounting and diagnostics.
CREATE TABLE download_attempts (
    id INTEGER PRIMARY KEY,
    run_id INTEGER REFERENCES download_runs(id) ON DELETE SET NULL,
    source_id INTEGER REFERENCES sources(id) ON DELETE SET NULL,
    video_id TEXT NOT NULL,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    status TEXT NOT NULL CHECK (status IN ('started', 'succeeded', 'failed')),
    started_at TEXT NOT NULL,
    finished_at TEXT,
    error_code TEXT,
    error_message TEXT
);

CREATE INDEX idx_attempts_started ON download_attempts (started_at);
CREATE INDEX idx_attempts_source_started ON download_attempts (source_id, started_at);

CREATE TABLE worker_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    is_paused INTEGER NOT NULL DEFAULT 0 CHECK (is_paused IN (0, 1)),
    updated_at TEXT NOT NULL
);

INSERT INTO worker_state (id, is_paused, updated_at)
VALUES (1, 0, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
