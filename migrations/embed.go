package migrations

import "embed"

// Files embeds all SQL migration files so the binary can run migrations on startup.
//
//go:embed *.sql
var Files embed.FS
