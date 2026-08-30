package mcp

import (
	"fmt"
	"strings"
)

func RegisterPrompts(server *Server) error {
	if err := server.RegisterPrompt(Prompt{
		Name:        "tailor_resume",
		Description: "Creates a tailored resume from a job description using the candidate profile and configured resume format.",
		Arguments: []PromptArgument{
			{Name: "job_description", Description: "Full job description (overview, requirements, and responsibilities).", Required: true},
			{Name: "file_name", Description: "Optional output file name without an extension.", Required: false},
		},
	}, tailorResumePrompt); err != nil {
		return err
	}

	return nil
}

func tailorResumePrompt(args map[string]string) (GetPromptResult, error) {
	jobDescription := strings.TrimSpace(args["job_description"])
	if jobDescription == "" {
		return GetPromptResult{}, fmt.Errorf("the 'job_description' argument cannot be empty")
	}
	if len(jobDescription) > 2<<20 {
		return GetPromptResult{}, fmt.Errorf("the 'job_description' argument is too long")
	}

	fileName := strings.TrimSpace(args["file_name"])
	if fileName != "" && (strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") || strings.Contains(fileName, "..") || strings.HasPrefix(fileName, ".")) {
		return GetPromptResult{}, fmt.Errorf("the 'file_name' argument must be a simple file name without paths")
	}
	fileInstruction := "Call save_resume without file_name so the server generates one."
	if fileName != "" {
		fileInstruction = fmt.Sprintf("Call save_resume with file_name set to %q.", fileName)
	}

	text := fmt.Sprintf("Create a tailored resume for the following job description.\n\n"+
		"1. Call prepare_job_context with the complete job description below.\n"+
		"2. Use only factual information from the returned candidate profile.\n"+
		"3. Default language rule: if the user does not explicitly request another language, write the resume in the profile's predominant language. Spanish when Spanish content is the majority, or English when English content is the majority. Never default to English just because the job description is in English.\n"+
		"4. Follow the returned resume format and use the profile's predominant language across headings and evaluation text.\n"+
		"5. Include relevant experience, skills, achievements, and a fit evaluation from 1 to 10.\n"+
		"6. Clearly identify weaknesses or gaps without inventing experience.\n"+
		"7. Write the final resume in Markdown.\n"+"8. %s\n\n"+
		"Job description:\n```\n%s\n```", fileInstruction, jobDescription)

	return GetPromptResult{
		Description: "Creates a tailored resume for a job description.",
		Messages: []PromptMessage{{
			Role:    "user",
			Content: ContentBlock{Type: "text", Text: text},
		}},
	}, nil
}
