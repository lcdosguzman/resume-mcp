package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("when environment is empty then Load returns default values", func(t *testing.T) {
		t.Setenv("MCP_PORT", "")
		t.Setenv("MCP_PUBLIC_URL", "")

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, defaultPort, cfg.Port)
		assert.Equal(t, "http://127.0.0.1:8090", cfg.PublicURL)
		assert.Equal(t, "data/profile.json", cfg.ProfilePath)
		assert.Equal(t, "data/resume_format.json", cfg.ResumeFormatPath)
		assert.Equal(t, "output", cfg.OutputDir)
	})

	t.Run("when runtime paths are valid then Validate succeeds", func(t *testing.T) {
		tempDir := t.TempDir()
		profilePath := filepath.Join(tempDir, "profile.json")
		formatPath := filepath.Join(tempDir, "format.json")
		outputDir := filepath.Join(tempDir, "output")

		require.NoError(t, os.WriteFile(profilePath, []byte("{}"), 0o600))
		require.NoError(t, os.WriteFile(formatPath, []byte("{}"), 0o600))

		cfg := Config{
			Port:             8090,
			PublicURL:        "http://127.0.0.1:8090",
			ProfilePath:      profilePath,
			ResumeFormatPath: formatPath,
			OutputDir:        outputDir,
		}

		require.NoError(t, cfg.Validate())
		assert.DirExists(t, outputDir)
	})

	t.Run("when runtime paths are invalid then Validate returns an error", func(t *testing.T) {
		cfg := Config{
			Port:             8090,
			PublicURL:        "http://127.0.0.1:8090",
			ProfilePath:      filepath.Join(t.TempDir(), "missing-profile.json"),
			ResumeFormatPath: filepath.Join(t.TempDir(), "missing-format.json"),
			OutputDir:        filepath.Join(t.TempDir(), "output"),
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ProfilePath")
	})

	t.Run("when custom values are set then Load returns configured values", func(t *testing.T) {
		t.Setenv("MCP_PORT", "9000")
		t.Setenv("MCP_PUBLIC_URL", "https://example.com/base/")

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, 9000, cfg.Port)
		assert.Equal(t, "https://example.com/base", cfg.PublicURL)
		assert.Equal(t, "127.0.0.1:9000", cfg.Address())
		assert.Equal(t, "https://example.com/base/downloads/", cfg.DownloadBaseURL())
	})

	t.Run("when invalid values are set then Load returns an error", func(t *testing.T) {
		cases := []struct {
			name    string
			port    string
			public  string
			wantErr string
		}{
			{name: "port is not numeric", port: "abc", public: "http://127.0.0.1:8090", wantErr: "invalid MCP_PORT"},
			{name: "port is out of range", port: "70000", public: "http://127.0.0.1:8090", wantErr: "invalid MCP_PORT"},
			{name: "public URL has no scheme", port: "8090", public: "example.com", wantErr: "invalid MCP_PUBLIC_URL"},
			{name: "public URL is malformed", port: "8090", public: "://bad", wantErr: "invalid MCP_PUBLIC_URL"},
		}

		for _, tc := range cases {
			t.Run("when "+tc.name+" then Load returns an error", func(t *testing.T) {
				t.Setenv("MCP_PORT", tc.port)
				t.Setenv("MCP_PUBLIC_URL", tc.public)

				_, err := Load()
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			})
		}
	})
}
