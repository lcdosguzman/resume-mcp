package resume

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Writer struct {
	outputDir       string
	downloadBaseURL string
	now             func() time.Time
}

func NewWriter(outputDir, downloadBaseURL string) Writer {
	return Writer{
		outputDir:       outputDir,
		downloadBaseURL: downloadBaseURL,
	}
}

var safeFileNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (w Writer) Save(content, requestedName string) (string, error) {
	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create output directory: %w", err)
	}

	fileName := strings.TrimSpace(requestedName)
	if fileName == "" {
		fileName = "cv"
	} else {
		fileName = safeFileNamePattern.ReplaceAllString(fileName, "_")
	}
	fileName = strings.Trim(fileName, "_")
	if fileName == "" {
		fileName = "cv"
	}

	clock := w.now
	if clock == nil {
		clock = time.Now
	}
	fileName += "_" + clock().Format("20060102_150405.000000000")

	path := filepath.Join(w.outputDir, fileName+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("could not write file: %w", err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("could not resolve saved file path: %w", err)
	}
	return fmt.Sprintf("Resume saved at: %s\nDownload: %s", abs, w.downloadBaseURL+url.PathEscape(filepath.Base(path))), nil
}
