package resume

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDataSource struct {
	profileData []byte
	formatData  []byte
	profileErr  error
	formatErr   error
}

func (s stubDataSource) ReadProfile() ([]byte, error) {
	if s.profileErr != nil {
		return nil, s.profileErr
	}
	return s.profileData, nil
}

func (s stubDataSource) ReadResumeFormat() ([]byte, error) {
	if s.formatErr != nil {
		return nil, s.formatErr
	}
	return s.formatData, nil
}

func TestPrepareJobContext(t *testing.T) {
	t.Run("when job description is empty then PrepareJobContext returns an error", func(t *testing.T) {
		service := NewService(stubDataSource{}, Writer{})

		_, err := service.PrepareJobContext("   \n\t  ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job_description")
	})

	t.Run("when profile and format are available then PrepareJobContext builds the job context", func(t *testing.T) {
		data := stubDataSource{
			profileData: []byte(`{"summary":"I am a Spanish-speaking Go engineer with AWS experience."}`),
			formatData:  []byte(`{"sections":["summary","experience"]}`),
		}
		service := NewService(data, Writer{})

		context, err := service.PrepareJobContext("We are hiring a Go backend engineer in AWS.")
		require.NoError(t, err)

		for _, needle := range []string{
			"# Context for generating a tailored resume",
			"## Job description",
			"## Candidate profile",
			"## Required resume format",
			"I am a Spanish-speaking Go engineer with AWS experience.",
			"{\"sections\":[\"summary\",\"experience\"]}",
		} {
			assert.Contains(t, context, needle)
		}
	})

	t.Run("when reading the data source fails then PrepareJobContext returns the read error", func(t *testing.T) {
		service := NewService(stubDataSource{profileErr: os.ErrNotExist, formatErr: os.ErrPermission}, Writer{})

		_, err := service.PrepareJobContext("Go developer")
		require.Error(t, err)
	})
}

func TestWriterHelpers(t *testing.T) {
	t.Run("when sanitizing a file name then invalid characters are normalized", func(t *testing.T) {
		writer := NewWriter(t.TempDir(), "https://example.com/downloads/")

		assert.Equal(t, "cv", writer.sanitizeFileName("   "))
		assert.Equal(t, "My_Resume_2024", writer.sanitizeFileName("My Resume 2024"))
		assert.Equal(t, "resume", writer.sanitizeFileName("_resume_"))
		assert.Equal(t, "go_backend", writer.sanitizeFileName("go/backend?"))
		assert.Equal(t, "etc_passwd", writer.sanitizeFileName("../../etc/passwd"))
	})

	t.Run("when file name exceeds the maximum safe length then it is truncated", func(t *testing.T) {
		writer := NewWriter(t.TempDir(), "https://example.com/downloads/")
		longName := strings.Repeat("a", maxSanitizedFileNameLength+50)

		assert.Len(t, writer.sanitizeFileName(longName), maxSanitizedFileNameLength)
	})

	t.Run("when generating a path then a collision creates a unique candidate", func(t *testing.T) {
		dir := t.TempDir()
		writer := NewWriter(dir, "https://example.com/downloads/")
		writer.now = func() time.Time {
			return time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		}

		basePath := filepath.Join(dir, "resume_20240102_030405.000000000.md")
		require.NoError(t, os.WriteFile(basePath, []byte("existing"), 0o644))

		path, err := writer.uniquePath("resume")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "resume_20240102_030405.000000000_1.md"), path)
	})

	t.Run("when writing a file then content is stored exactly as provided", func(t *testing.T) {
		dir := t.TempDir()
		writer := NewWriter(dir, "https://example.com/downloads/")
		path := filepath.Join(dir, "sample.md")

		err := writer.writeFile(path, "# Hello\n")
		require.NoError(t, err)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "# Hello\n", string(content))
	})

	t.Run("when building a saved message then it contains the absolute path and download URL", func(t *testing.T) {
		dir := t.TempDir()
		writer := NewWriter(dir, "https://example.com/downloads/")
		path := filepath.Join(dir, "resume_20240102_030405.000000000.md")

		message, err := writer.savedMessage(path)
		require.NoError(t, err)
		assert.Contains(t, message, "Resume saved at:")
		assert.Contains(t, message, "Download: https://example.com/downloads/resume_20240102_030405.000000000.md")
	})
}

func TestSave(t *testing.T) {
	t.Run("when content is empty then Save returns an error", func(t *testing.T) {
		service := NewService(stubDataSource{}, NewWriter(t.TempDir(), "https://example.com/downloads"))

		_, err := service.Save("   ", "resume")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content")
	})

	t.Run("when file name contains path traversal then Save rejects it", func(t *testing.T) {
		service := NewService(stubDataSource{}, NewWriter(t.TempDir(), "https://example.com/downloads"))

		_, err := service.Save("# Hello\n", "../../evil")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file_name")
	})

	t.Run("when content exceeds the maximum size then Save returns an error", func(t *testing.T) {
		service := NewService(stubDataSource{}, NewWriter(t.TempDir(), "https://example.com/downloads"))
		largeContent := strings.Repeat("A", maxResumeContentSize+1)

		_, err := service.Save(largeContent, "resume")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content")
	})

	t.Run("when content and file name are valid then Save writes the markdown file", func(t *testing.T) {
		dir := t.TempDir()
		writer := NewWriter(dir, "https://example.com/downloads/")
		writer.now = func() time.Time {
			return time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		}
		service := NewService(stubDataSource{}, writer)

		message, err := service.Save("# Hello\n", "My Resume 2024")
		require.NoError(t, err)
		assert.Contains(t, message, "Resume saved at:")
		assert.Contains(t, message, "Download: https://example.com/downloads/")

		matches, err := filepath.Glob(filepath.Join(dir, "My_Resume_2024_20240102_030405.000000000*.md"))
		require.NoError(t, err)
		assert.Len(t, matches, 1)

		content, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		assert.Equal(t, "# Hello\n", string(content))
	})

	t.Run("when a file with the same name already exists then Save creates a unique file name", func(t *testing.T) {
		dir := t.TempDir()
		writer := NewWriter(dir, "https://example.com/downloads/")
		writer.now = func() time.Time {
			return time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		}
		service := NewService(stubDataSource{}, writer)

		firstPath := filepath.Join(dir, "resume_20240102_030405.000000000.md")
		require.NoError(t, os.WriteFile(firstPath, []byte("existing"), 0o644))

		message, err := service.Save("# Hello\n", "resume")
		require.NoError(t, err)
		assert.Contains(t, message, "Resume saved at:")

		matches, err := filepath.Glob(filepath.Join(dir, "resume_20240102_030405.000000000*.md"))
		require.NoError(t, err)
		assert.Len(t, matches, 2)
	})
}
