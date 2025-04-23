//go:build !prod
// +build !prod

package main

import "net/http"

func addStatic(mux *http.ServeMux) {
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
}
