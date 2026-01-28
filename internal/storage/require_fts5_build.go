//go:build !fts5

package storage

// This file only compiles when the "fts5" build tag is missing. The undefined
// identifier below forces a compile-time failure so tinymem cannot be built
// without SQLite FTS5 support. Rebuild with `-tags fts5` to satisfy the
// requirement.
const _ = requireFTS5BuildTag
