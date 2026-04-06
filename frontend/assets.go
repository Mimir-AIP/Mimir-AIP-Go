package frontendassets

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

// Assets contains the packaged frontend files used by the local all-in-one launcher.
//
//go:embed index.html styles.css app.js lib context hooks components pages vendor
var Assets embed.FS

func Handler() http.Handler {
	subtree, err := fs.Sub(Assets, ".")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(subtree))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean(r.URL.Path)
		if cleanPath == "." || cleanPath == "/" {
			http.ServeFileFS(w, r, subtree, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
