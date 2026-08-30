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

const maxSanitizedFileNameLength = 80

var safeFileNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (w Writer) Save(content, requestedName string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("the 'content' field cannot be empty")
	}
	if len(content) > maxResumeContentSize {
		return "", fmt.Errorf("the 'content' field is too long")
	}
	if strings.Contains(requestedName, "/") || strings.Contains(requestedName, "\\") || strings.Contains(requestedName, "..") || strings.HasPrefix(requestedName, ".") {
		return "", fmt.Errorf("the 'file_name' field must be a simple file name without paths")
	}
	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create output directory: %w", err)
	}

	fileName := w.sanitizeFileName(requestedName)
	path, err := w.uniquePath(fileName)
	if err != nil {
		return "", err
	}
	if err := w.writeFile(path, content); err != nil {
		return "", err
	}
	return w.savedMessage(path)
}

func (w Writer) sanitizeFileName(requestedName string) string {
	fileName := strings.TrimSpace(requestedName)
	if fileName == "" {
		return "cv"
	}

	fileName = strings.ReplaceAll(fileName, "\\", "/")
	fileName = filepath.Clean(fileName)
	fileName = strings.Trim(fileName, "/.")
	fileName = safeFileNamePattern.ReplaceAllString(fileName, "_")
	fileName = strings.Trim(fileName, "_")
	if fileName == "" {
		return "cv"
	}
	if len(fileName) > maxSanitizedFileNameLength {
		fileName = fileName[:maxSanitizedFileNameLength]
		fileName = strings.TrimRight(fileName, "_")
	}
	if fileName == "" {
		return "cv"
	}
	return fileName
}

func (w Writer) uniquePath(fileName string) (string, error) {
	clock := w.now
	if clock == nil {
		clock = time.Now
	}
	baseName := fileName + "_" + clock().Format("20060102_150405.000000000")
	path := filepath.Join(w.outputDir, baseName+".md")

	for i := 0; ; i++ {
		if _, err := os.Stat(path); err == nil {
			path = filepath.Join(w.outputDir, fmt.Sprintf("%s_%d.md", baseName, i+1))
			continue
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("could not check file existence: %w", err)
		}
		return path, nil
	}
}

func (w Writer) writeFile(path, content string) error {
	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("could not write file: %w", err)
		}
		return fmt.Errorf("could not write file: %w", err)
	}
	defer fd.Close()

	if _, err := fd.Write([]byte(content)); err != nil {
		return fmt.Errorf("could not write file: %w", err)
	}
	return nil
}

func (w Writer) savedMessage(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("could not resolve saved file path: %w", err)
	}
	return fmt.Sprintf("Resume saved at: %s\nDownload: %s", abs, w.downloadBaseURL+url.PathEscape(filepath.Base(path))), nil
}
