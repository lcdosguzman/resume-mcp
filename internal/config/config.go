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

func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid Port %d", c.Port)
	}
	if c.PublicURL == "" {
		return fmt.Errorf("invalid PublicURL %q", c.PublicURL)
	}
	parsedURL, err := url.Parse(c.PublicURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("invalid PublicURL %q", c.PublicURL)
	}
	if strings.TrimSpace(c.ProfilePath) == "" {
		return fmt.Errorf("invalid ProfilePath %q", c.ProfilePath)
	}
	if strings.TrimSpace(c.ResumeFormatPath) == "" {
		return fmt.Errorf("invalid ResumeFormatPath %q", c.ResumeFormatPath)
	}
	if strings.TrimSpace(c.OutputDir) == "" {
		return fmt.Errorf("invalid OutputDir %q", c.OutputDir)
	}

	if _, err := os.Stat(c.ProfilePath); err != nil {
		return fmt.Errorf("invalid ProfilePath %q: %w", c.ProfilePath, err)
	}
	if _, err := os.Stat(c.ResumeFormatPath); err != nil {
		return fmt.Errorf("invalid ResumeFormatPath %q: %w", c.ResumeFormatPath, err)
	}

	if err := os.MkdirAll(c.OutputDir, 0o755); err != nil {
		return fmt.Errorf("invalid OutputDir %q: %w", c.OutputDir, err)
	}

	info, err := os.Stat(c.OutputDir)
	if err != nil {
		return fmt.Errorf("invalid OutputDir %q: %w", c.OutputDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("invalid OutputDir %q: path is not a directory", c.OutputDir)
	}
	return nil
}

func filepathBase(path string) string {
	base := path
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	if idx := strings.LastIndex(base, "\\"); idx >= 0 {
		base = base[idx+1:]
	}
	return base
}
