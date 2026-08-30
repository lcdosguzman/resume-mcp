package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	t.Run("when profile and format files exist then ReadProfile and ReadResumeFormat return data", func(t *testing.T) {
		dir := t.TempDir()
		profilePath := filepath.Join(dir, "profile.json")
		formatPath := filepath.Join(dir, "resume_format.json")

		profileData := []byte(`{"name":"Ada"}`)
		formatData := []byte(`{"sections":["summary"]}`)

		require.NoError(t, os.WriteFile(profilePath, profileData, 0o600))
		require.NoError(t, os.WriteFile(formatPath, formatData, 0o600))

		repo := NewRepository(profilePath, formatPath)

		gotProfile, err := repo.ReadProfile()
		require.NoError(t, err)
		assert.Equal(t, string(profileData), string(gotProfile))

		gotFormat, err := repo.ReadResumeFormat()
		require.NoError(t, err)
		assert.Equal(t, string(formatData), string(gotFormat))
	})

	t.Run("when repository files are missing then reads return errors", func(t *testing.T) {
		cases := []struct {
			name     string
			readFunc func(Repository) ([]byte, error)
		}{
			{name: "profile is missing", readFunc: func(repo Repository) ([]byte, error) { return repo.ReadProfile() }},
			{name: "format is missing", readFunc: func(repo Repository) ([]byte, error) { return repo.ReadResumeFormat() }},
		}

		for _, tc := range cases {
			t.Run("when "+tc.name+" then repository returns an error", func(t *testing.T) {
				dir := t.TempDir()
				repo := NewRepository(filepath.Join(dir, "missing_profile.json"), filepath.Join(dir, "missing_format.json"))

				_, err := tc.readFunc(repo)
				require.Error(t, err)
				assert.NotEmpty(t, err.Error())
			})
		}
	})
}
