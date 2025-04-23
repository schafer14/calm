//go:build prod
// +build prod

package main

import "net/http"
import "github.com/gorilla/handlers"

func addStatic(mux *http.ServeMux) {
	mux.Handle("/static/", handlers.CompressHandler(http.FileServer(http.FS(static))))
}
