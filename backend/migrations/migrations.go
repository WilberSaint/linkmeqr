// Package migrations embeds the SQL schema files into the binary, so a
// deploy is just the executable — no migrations/ folder to keep beside it.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
