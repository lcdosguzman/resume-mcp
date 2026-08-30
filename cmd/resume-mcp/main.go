package main

import (
	"log"
	"net/http"

	"resume-mcp/internal/config"
	"resume-mcp/internal/profile"
	"resume-mcp/internal/resume"
	mcpserver "resume-mcp/mcp"
)

func main() {
	appConfig, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	profileRepository := profile.NewRepository(appConfig.ProfilePath, appConfig.ResumeFormatPath)
	resumeWriter := resume.NewWriter(appConfig.OutputDir, appConfig.DownloadBaseURL())
	resumeService := resume.NewService(profileRepository, resumeWriter)

	mcpServer := mcpserver.NewServer("resume-mcp", "0.1.0")
	if err := mcpserver.RegisterTools(mcpServer, profileRepository, resumeService); err != nil {
		log.Fatalf("could not register tools: %v", err)
	}
	if err := mcpserver.RegisterPrompts(mcpServer); err != nil {
		log.Fatalf("could not register prompts: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpServer.Handler())
	mux.Handle("/downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir(appConfig.OutputDir))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	address := appConfig.Address()
	log.Printf("resume-mcp listening at http://%s/mcp (Ctrl+C to stop)", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
