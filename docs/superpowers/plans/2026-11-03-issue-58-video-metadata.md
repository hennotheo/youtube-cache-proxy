# Issue 58 — Enrichir les métadonnées vidéo de l’API

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Return persisted channel, thumbnail, publication, description, tags, categories, view count, language, and live status metadata on source creation/list/detail without downloading audio.

**Architecture:** Extend the existing yt-dlp metadata-only query and map its optional fields into a stable video metadata model. Persist nullable scalar fields and JSON arrays through a forward-only SQLite migration, then expose them through the existing source DTOs. Metadata lookup failure leaves the new source pending and does not fail source creation.

**Tech Stack:** Go 1.27, standard `net/http`, yt-dlp JSON output, SQLite migrations.

**Spec:** GitHub issue #58.

## Global Constraints

- Keep metadata extraction metadata-only; do not start audio downloads when adding a source.
- Use the validated canonical video ID/URL and never invoke yt-dlp through a shell or cookies.
- Missing optional values must remain absent/null; old SQLite rows remain readable.
- Keep one struct per file and update the README when app behavior changes.

## Review Focus

- yt-dlp omits optional fields or returns malformed dates: source creation still succeeds and absent values remain null.
- Source is deleted or its state changes while metadata lookup runs: update must not corrupt unrelated state.
- Existing databases and records survive the new migration.
- Tags/categories contain Unicode, commas, or empty lists and round-trip as JSON arrays.
- Metadata lookup command is canceled or fails: no audio file is downloaded and the pending source remains.

## Tasks

### Task 1: Map metadata-only yt-dlp JSON

**Files:** `internal/youtube/yt_dlp_metadata.go`, `video_metadata.go`, `metadata.go`, `metadata_args.go`, `downloader.go`, `downloader_test.go`.

- [ ] Add a fake-process test for extended JSON parsing and verify `--skip-download` with no audio download arguments.
- [ ] Add `Downloader.FetchMetadata(ctx, videoID)` and map optional fields; retain the existing strict duration check for actual downloads.
- [ ] Verify the focused YouTube tests.

### Task 2: Persist metadata in SQLite

**Files:** new migration, `internal/store/source.go`, `scan_source.go`, `complete_attempt.go`, new metadata persistence file, store tests.

- [ ] Test migration compatibility and metadata round-trip, including tags/categories.
- [ ] Add nullable columns and `Store.UpdateSourceMetadata`; persist full metadata after downloads as fallback.
- [ ] Verify the focused store tests.

### Task 3: Add metadata to source API responses

**Files:** `internal/httpapi/api.go`, `sources.go`, `dto.go`, new API tests, `cmd/server/main.go`.

- [ ] Test `POST /sources` returns saved metadata while status remains pending and still creates a source when metadata lookup fails.
- [ ] Inject the metadata provider, persist successful lookups, and expose the same fields in create/list/detail responses.
- [ ] Verify the focused HTTP API tests.

### Task 4: Document and verify

**Files:** `README.md`.

- [ ] Document metadata fields, nullable behavior, and metadata-only lookup at source creation.
- [ ] Run `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check`.
- [ ] Commit implementation and README in separate atomic commits.
