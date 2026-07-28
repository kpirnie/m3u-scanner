package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/index.html static/admin.css static/admin.js
var staticFS embed.FS

// adminAssets returns the embedded admin interface asset tree rooted at the
// static directory, so requests address files without a /static/ prefix.
func adminAssets() http.FileSystem {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
