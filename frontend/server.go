package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Serve static files from the frontend directory
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)

	log.Printf("Frontend server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
