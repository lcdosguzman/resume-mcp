package main

import (
	"log"
	"net/http"

	mcpserver "resume-mcp/mcp"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	srv := mcpserver.NewServer("resume-mcp", "0.1.0")
	app := newApplication(config)
	registerTools(srv, app)
	registerPrompts(srv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", srv.Handler())
	mux.Handle("/downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir(config.outputDir))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	addr := config.address()
	log.Printf("resume-mcp listening at http://%s/mcp (Ctrl+C to stop)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
