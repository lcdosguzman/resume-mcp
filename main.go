package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	mcpserver "resume-mcp/mcp"
)

const (
	defaultPort      = "8090"
	profilePath      = "data/profile.json"
	resumeFormatPath = "data/resume_format.json"
	outputDir        = "output"
)

func main() {
	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = defaultPort
	}
	publicURL := os.Getenv("MCP_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://127.0.0.1:" + port
	}
	downloadBaseURL = strings.TrimRight(publicURL, "/") + "/downloads/"

	srv := mcpserver.NewServer("resume-mcp", "0.1.0")
	registerTools(srv)

	mux := http.NewServeMux()
	mux.Handle("/mcp", srv.Handler())
	mux.Handle("/downloads/", http.StripPrefix("/downloads/", http.FileServer(http.Dir(outputDir))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	addr := "127.0.0.1:" + port
	log.Printf("resume-mcp listening at http://%s/mcp (Ctrl+C to stop)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}

func registerTools(srv *mcpserver.Server) {
	srv.RegisterTool(mcpserver.Tool{
		Name:        "get_profile",
		Description: "Returns the user's personal details, experience, education, and skills as JSON. Use this first to review the available information before tailoring a resume.",
		InputSchema: mcpserver.InputSchema{Type: "object"},
	}, getProfileTool)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "get_resume_format",
		Description: "Returns the structure and format that the generated resume must follow (sections, order, style, and length). Use it together with get_profile.",
		InputSchema: mcpserver.InputSchema{Type: "object"},
	}, getResumeFormatTool)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "prepare_job_context",
		Description: "Receives a job description and returns instructions for generating a resume tailored to that position, together with the profile and resume format. The model calling this tool must use the result to write the resume in Markdown and then save it with the 'save_resume' tool.",
		InputSchema: mcpserver.InputSchema{
			Type: "object",
			Properties: map[string]mcpserver.Property{
				"job_description": {Type: "string", Description: "Full job description (overview, requirements, and responsibilities)."},
			},
			Required: []string{"job_description"},
		},
	}, prepareJobContextTool)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "save_resume",
		Description: "Saves the tailored resume, already written in Markdown by the model, as an .md file in the server's output/ directory.",
		InputSchema: mcpserver.InputSchema{
			Type: "object",
			Properties: map[string]mcpserver.Property{
				"content":   {Type: "string", Description: "Complete resume content in Markdown format."},
				"file_name": {Type: "string", Description: "Optional file name without an extension. If omitted, one is generated with the current date and time."},
			},
			Required: []string{"content"},
		},
	}, saveResumeTool)
}

// ---------- Handlers ----------

func getProfileTool(_ json.RawMessage) (mcpserver.CallToolResult, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return errResult(fmt.Errorf("could not read %s: %w", profilePath, err))
	}
	return textResult(string(data)), nil
}

func getResumeFormatTool(_ json.RawMessage) (mcpserver.CallToolResult, error) {
	data, err := os.ReadFile(resumeFormatPath)
	if err != nil {
		return errResult(fmt.Errorf("could not read %s: %w", resumeFormatPath, err))
	}
	return textResult(string(data)), nil
}

type prepareJobArgs struct {
	JobDescription string `json:"job_description"`
}

func prepareJobContextTool(raw json.RawMessage) (mcpserver.CallToolResult, error) {
	var args prepareJobArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return errResult(fmt.Errorf("invalid arguments: %w", err))
	}
	if strings.TrimSpace(args.JobDescription) == "" {
		return errResult(fmt.Errorf("the 'job_description' field cannot be empty"))
	}

	profile, err := os.ReadFile(profilePath)
	if err != nil {
		return errResult(fmt.Errorf("could not read %s: %w", profilePath, err))
	}
	format, err := os.ReadFile(resumeFormatPath)
	if err != nil {
		return errResult(fmt.Errorf("could not read %s: %w", resumeFormatPath, err))
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

var safeFileNamePattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
var downloadBaseURL string

func saveResumeTool(raw json.RawMessage) (mcpserver.CallToolResult, error) {
	var args saveResumeArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return errResult(fmt.Errorf("invalid arguments: %w", err))
	}
	if strings.TrimSpace(args.ResumeContent) == "" {
		return errResult(fmt.Errorf("the 'content' field cannot be empty"))
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return errResult(fmt.Errorf("could not create output directory: %w", err))
	}

	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		fileName = "cv"
	} else {
		fileName = safeFileNamePattern.ReplaceAllString(fileName, "_")
	}
	fileName = strings.Trim(fileName, "_")
	if fileName == "" {
		fileName = "cv"
	}
	fileName += "_" + time.Now().Format("20060102_150405")

	path := filepath.Join(outputDir, fileName+".md")
	if err := os.WriteFile(path, []byte(args.ResumeContent), 0o644); err != nil {
		return errResult(fmt.Errorf("could not write file: %w", err))
	}

	abs, _ := filepath.Abs(path)
	downloadURL := downloadBaseURL + url.PathEscape(filepath.Base(path))
	return textResult(fmt.Sprintf("Resume saved at: %s\nDownload: %s", abs, downloadURL)), nil
}

// ---------- Helpers ----------

func textResult(text string) mcpserver.CallToolResult {
	return mcpserver.CallToolResult{
		Content: []mcpserver.ContentBlock{{Type: "text", Text: text}},
	}
}

func errResult(err error) (mcpserver.CallToolResult, error) {
	return mcpserver.CallToolResult{}, err
}
