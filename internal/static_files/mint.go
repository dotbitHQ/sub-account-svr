package static_files

import (
	"embed"
	_ "embed"
)

//go:embed mint.js
var MintJs embed.FS
