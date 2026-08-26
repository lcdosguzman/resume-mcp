package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	mcpserver "resume-mcp/mcp"
)

type application struct {
	data   fileStore
	resume resumeWriter
}

func newApplication(config config) application {
	return application{
		data: fileStore{
			profilePath:      config.profilePath,
			resumeFormatPath: config.resumeFormatPath,
		},
		resume: resumeWriter{
			outputDir:       config.outputDir,
			downloadBaseURL: config.downloadBaseURL(),
		},
	}
}

type fileStore struct {
	profilePath      string
	resumeFormatPath string
}

func (s fileStore) readProfile() ([]byte, error) {
	data, err := os.ReadFile(s.profilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", s.profilePath, err)
	}
	return data, nil
}

func (s fileStore) readResumeFormat() ([]byte, error) {
	data, err := os.ReadFile(s.resumeFormatPath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", s.resumeFormatPath, err)
	}
	return data, nil
}

type resumeWriter struct {
	outputDir       string
	downloadBaseURL string
	now             func() time.Time
}

var safeFileNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (w resumeWriter) save(content, requestedName string) (string, error) {
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

func (app application) getProfileTool(_ json.RawMessage) (mcpserver.CallToolResult, error) {
	data, err := app.data.readProfile()
	if err != nil {
		return errResult(err)
	}
	return textResult(string(data)), nil
}

func (app application) getResumeFormatTool(_ json.RawMessage) (mcpserver.CallToolResult, error) {
	data, err := app.data.readResumeFormat()
	if err != nil {
		return errResult(err)
	}
	return textResult(string(data)), nil
}

type prepareJobArgs struct {
	JobDescription string `json:"job_description"`
}

func (app application) prepareJobContextTool(raw json.RawMessage) (mcpserver.CallToolResult, error) {
	var args prepareJobArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return errResult(fmt.Errorf("invalid arguments: %w", err))
	}
	if strings.TrimSpace(args.JobDescription) == "" {
		return errResult(fmt.Errorf("the 'job_description' field cannot be empty"))
	}

	profile, err := app.data.readProfile()
	if err != nil {
		return errResult(err)
	}
	format, err := app.data.readResumeFormat()
	if err != nil {
		return errResult(err)
	}

	var sb strings.Builder
	sb.WriteString("# Context for generating a tailored resume\n\n")
	sb.WriteString("## Instructions\n")
	sb.WriteString("First determine the predominant language of the profile's narrative content. Compare the prose in the summary, descriptions, achievements, education, certifications, and soft skills; do not count only names, company names, technologies, or URLs. Write the entire resume, including section headings and the fit evaluation, in the predominant language: Spanish when Spanish content is the majority, or English when English content is the majority. Then write a Markdown resume tailored to the job description below, using ")
	sb.WriteString("only factual information from the profile (do not invent experience, ")
	sb.WriteString("degrees, or dates). Prioritize and reorder the experience, achievements, and skills ")
	sb.WriteString("that are most relevant to this specific position. Follow the structure defined ")
	sb.WriteString("in 'resume_format'. The final fit evaluation section is mandatory and must include ")
	sb.WriteString("an integer score from 1 to 10, strengths, and weaknesses or gaps, ")
	sb.WriteString("clearly distinguishing demonstrated experience from unverified experience. When finished, ")
	sb.WriteString("save the result using the 'save_resume' tool.\n\n")

	sb.WriteString("## Job description\n```\n")
	sb.WriteString(args.JobDescription)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Candidate profile (data/profile.json)\n```json\n")
	sb.Write(profile)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Required resume format (resume_format.json)\n```json\n")
	sb.Write(format)
	sb.WriteString("\n```\n")

	return textResult(sb.String()), nil
}

type saveResumeArgs struct {
	ResumeContent string `json:"content"`
	FileName      string `json:"file_name"`
}

func (app application) saveResumeTool(raw json.RawMessage) (mcpserver.CallToolResult, error) {
	var args saveResumeArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return errResult(fmt.Errorf("invalid arguments: %w", err))
	}
	if strings.TrimSpace(args.ResumeContent) == "" {
		return errResult(fmt.Errorf("the 'content' field cannot be empty"))
	}

	message, err := app.resume.save(args.ResumeContent, args.FileName)
	if err != nil {
		return errResult(err)
	}
	return textResult(message), nil
}

func textResult(text string) mcpserver.CallToolResult {
	return mcpserver.CallToolResult{
		Content: []mcpserver.ContentBlock{{Type: "text", Text: text}},
	}
}

func errResult(err error) (mcpserver.CallToolResult, error) {
	return mcpserver.CallToolResult{}, err
}
