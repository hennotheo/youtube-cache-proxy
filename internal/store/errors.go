package store

import "errors"

var (
	// ErrDuplicate reports that a video already exists in the queue.
	ErrDuplicate = errors.New("source already exists")
	// ErrNotFound reports that a requested record does not exist.
	ErrNotFound = errors.New("record not found")
	// ErrConflict reports an invalid state transition or active-run conflict.
	ErrConflict = errors.New("state conflict")
	// ErrNoPending reports that no pending source is available for the requested run.
	ErrNoPending = errors.New("no pending source")
	// ErrPaused reports that the worker is paused.
	ErrPaused = errors.New("worker paused")
	// ErrQuotaExceeded reports that a new attempt would exceed the rolling daily limit.
	ErrQuotaExceeded = errors.New("daily attempt quota exceeded")
)
