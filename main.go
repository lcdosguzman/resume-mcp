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
	registerPrompts(srv)

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

func registerPrompts(srv *mcpserver.Server) {
	srv.RegisterPrompt(mcpserver.Prompt{
		Name:        "tailor_resume",
		Description: "Creates a tailored resume from a job description using the candidate profile and configured resume format.",
		Arguments: []mcpserver.PromptArgument{
			{Name: "job_description", Description: "Full job description (overview, requirements, and responsibilities).", Required: true},
			{Name: "file_name", Description: "Optional output file name without an extension.", Required: false},
		},
	}, tailorResumePrompt)
}

func tailorResumePrompt(args map[string]string) (mcpserver.GetPromptResult, error) {
	jobDescription := strings.TrimSpace(args["job_description"])
	if jobDescription == "" {
		return mcpserver.GetPromptResult{}, fmt.Errorf("the 'job_description' argument cannot be empty")
	}

	fileName := strings.TrimSpace(args["file_name"])
	fileInstruction := "Call save_resume without file_name so the server generates one."
	if fileName != "" {
		fileInstruction = fmt.Sprintf("Call save_resume with file_name set to %q.", fileName)
	}

	text := fmt.Sprintf("Create a tailored resume for the following job description.\n\n"+
		"1. Call prepare_job_context with the complete job description below.\n"+
		"2. Use only factual information from the returned candidate profile.\n"+
		"3. Follow the returned resume format and use the profile's predominant language.\n"+
		"4. Include relevant experience, skills, achievements, and a fit evaluation from 1 to 10.\n"+
		"5. Clearly identify weaknesses or gaps without inventing experience.\n"+
		"6. Write the final resume in Markdown.\n"+"7. %s\n\n"+
		"Job description:\n```\n%s\n```", fileInstruction, jobDescription)

	return mcpserver.GetPromptResult{
		Description: "Creates a tailored resume for a job description.",
		Messages: []mcpserver.PromptMessage{{
			Role:    "user",
			Content: mcpserver.ContentBlock{Type: "text", Text: text},
		}},
	}, nil
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
