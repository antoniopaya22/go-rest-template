// Package migrations embeds the SQL migrations, one directory per database driver.
package migrations

import "embed"

// FS holds every migration file.
//
//go:embed */*.sql
var FS embed.FS
