package main

import (
	"context"
	"log"
	"net/http"
)

func main() {
	setupApi()

	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}

func setupApi() {
	manager := NewManager(context.Background())

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWs)
	http.HandleFunc("/login", manager.loginHandler)
}
