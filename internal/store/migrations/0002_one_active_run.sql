-- The database enforces the single manual batch invariant across processes.
CREATE UNIQUE INDEX idx_download_runs_one_active ON download_runs (status) WHERE status = 'active';
