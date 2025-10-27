package web

import "embed"

//go:embed static
var Static embed.FS

// Legacy exports for backward compatibility
//go:embed static/index.html
var IndexHTML []byte

//go:embed static/styles.css
var StylesCSS []byte
