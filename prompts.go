package main

import (
	"fmt"
	"strings"

	mcpserver "resume-mcp/mcp"
)

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
