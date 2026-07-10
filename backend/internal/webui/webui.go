// Package webui embeds the built frontend (frontend/dist, copied here by
// `make build`) so a release binary can serve the dashboard without a
// separate Vite dev server. dist/ is empty (just .gitkeep) in source
// control — go:embed only sees real content after the build step runs.
package webui

import "embed"

// "all:" is needed so the dist/.gitkeep placeholder (a dotfile, excluded by
// plain "embed dist") doesn't make this fail to compile pre-build.
//
//go:embed all:dist
var DistFS embed.FS
