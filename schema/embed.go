// Package schema embeds the JSON Schema files that define the muninn content
// format, so the binary can validate content and print the contract.
package schema

import "embed"

//go:embed *.schema.json
var Files embed.FS
