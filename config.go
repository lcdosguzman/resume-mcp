package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const defaultPort = 8090

type config struct {
	port             int
	publicURL        string
	profilePath      string
	resumeFormatPath string
	outputDir        string
}

func loadConfig() (config, error) {
	port := defaultPort
	if value := os.Getenv("MCP_PORT"); value != "" {
		parsedPort, err := strconv.Atoi(value)
		if err != nil || parsedPort < 1 || parsedPort > 65535 {
			return config{}, fmt.Errorf("invalid MCP_PORT %q", value)
		}
		port = parsedPort
	}

	publicURL := os.Getenv("MCP_PUBLIC_URL")
	if publicURL == "" {
		publicURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	parsedURL, err := url.Parse(publicURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return config{}, fmt.Errorf("invalid MCP_PUBLIC_URL %q", publicURL)
	}

	return config{
		port:             port,
		publicURL:        strings.TrimRight(publicURL, "/"),
		profilePath:      "data/profile.json",
		resumeFormatPath: "data/resume_format.json",
		outputDir:        "output",
	}, nil
}

func (c config) address() string {
	return fmt.Sprintf("127.0.0.1:%d", c.port)
}

func (c config) downloadBaseURL() string {
	return c.publicURL + "/downloads/"
}
