package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const defaultPort = 8090

type Config struct {
	Port             int
	PublicURL        string
	ProfilePath      string
	ResumeFormatPath string
	OutputDir        string
}

func Load() (Config, error) {
	port := defaultPort
	if value := os.Getenv("MCP_PORT"); value != "" {
		parsedPort, err := strconv.Atoi(value)
		if err != nil || parsedPort < 1 || parsedPort > 65535 {
			return Config{}, fmt.Errorf("invalid MCP_PORT %q", value)
		}
		port = parsedPort
	}

	publicURL := os.Getenv("MCP_PUBLIC_URL")
	if publicURL == "" {
		publicURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	parsedURL, err := url.Parse(publicURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return Config{}, fmt.Errorf("invalid MCP_PUBLIC_URL %q", publicURL)
	}

	return Config{
		Port:             port,
		PublicURL:        strings.TrimRight(publicURL, "/"),
		ProfilePath:      "data/profile.json",
		ResumeFormatPath: "data/resume_format.json",
		OutputDir:        "output",
	}, nil
}

func (c Config) Address() string {
	return fmt.Sprintf("127.0.0.1:%d", c.Port)
}

func (c Config) DownloadBaseURL() string {
	return c.PublicURL + "/downloads/"
}
