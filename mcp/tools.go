package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"resume-mcp/internal/profile"
	"resume-mcp/internal/resume"
)

func RegisterTools(server *Server, profileRepository profile.Repository, resumeService resume.Service) error {
	if err := server.RegisterTool(Tool{
		Name:        "get_profile",
		Description: "Returns the user's personal details, experience, education, and skills as JSON. Use this first to review the available information before tailoring a resume.",
		InputSchema: InputSchema{Type: "object"},
	}, getProfileTool(profileRepository)); err != nil {
		return err
	}

	if err := server.RegisterTool(Tool{
		Name:        "get_resume_format",
		Description: "Returns the structure and format that the generated resume must follow (sections, order, style, and length). Use it together with get_profile.",
		InputSchema: InputSchema{Type: "object"},
	}, getResumeFormatTool(profileRepository)); err != nil {
		return err
	}

	if err := server.RegisterTool(Tool{
		Name:        "prepare_job_context",
		Description: "Receives a job description and returns instructions for generating a resume tailored to that position, together with the profile and resume format. The model calling this tool must use the result to write the resume in Markdown and then save it with the 'save_resume' tool.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"job_description": {Type: "string", Description: "Full job description (overview, requirements, and responsibilities)."},
			},
			Required: []string{"job_description"},
		},
	}, prepareJobContextTool(resumeService)); err != nil {
		return err
	}

	if err := server.RegisterTool(Tool{
		Name:        "save_resume",
		Description: "Saves the tailored resume, already written in Markdown by the model, as an .md file in the server's output/ directory.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"content":   {Type: "string", Description: "Complete resume content in Markdown format."},
				"file_name": {Type: "string", Description: "Optional file name without an extension. If omitted, one is generated with the current date and time."},
			},
			Required: []string{"content"},
		},
	}, saveResumeTool(resumeService)); err != nil {
		return err
	}

	return nil
}

func getProfileTool(repository profile.Repository) ToolHandler {
	return func(_ json.RawMessage) (CallToolResult, error) {
		data, err := repository.ReadProfile()
		if err != nil {
			return errResult(err)
		}
		return textResult(string(data)), nil
	}
}

func getResumeFormatTool(repository profile.Repository) ToolHandler {
	return func(_ json.RawMessage) (CallToolResult, error) {
		data, err := repository.ReadResumeFormat()
		if err != nil {
			return errResult(err)
		}
		return textResult(string(data)), nil
	}
}

type prepareJobArgs struct {
	JobDescription string `json:"job_description"`
}

func prepareJobContextTool(service resume.Service) ToolHandler {
	return func(raw json.RawMessage) (CallToolResult, error) {
		var args prepareJobArgs
		if err := json.Unmarshal(raw, &args); err != nil {
			return errResult(fmt.Errorf("invalid arguments: %w", err))
		}
		if strings.TrimSpace(args.JobDescription) == "" {
			return errResult(fmt.Errorf("job_description is required"))
		}
		if len(args.JobDescription) > 2<<20 {
			return errResult(fmt.Errorf("job_description is too long"))
		}

		context, err := service.PrepareJobContext(args.JobDescription)
		if err != nil {
			return errResult(err)
		}
		return textResult(context), nil
	}
}

type saveResumeArgs struct {
	ResumeContent string `json:"content"`
	FileName      string `json:"file_name"`
}

func saveResumeTool(service resume.Service) ToolHandler {
	return func(raw json.RawMessage) (CallToolResult, error) {
		var args saveResumeArgs
		if err := json.Unmarshal(raw, &args); err != nil {
			return errResult(fmt.Errorf("invalid arguments: %w", err))
		}
		if strings.TrimSpace(args.ResumeContent) == "" {
			return errResult(fmt.Errorf("content is required"))
		}
		if args.FileName != "" {
			if strings.Contains(args.FileName, "/") || strings.Contains(args.FileName, "\\") || strings.Contains(args.FileName, "..") || strings.HasPrefix(args.FileName, ".") {
				return errResult(fmt.Errorf("file_name must be a simple file name without paths"))
			}
		}

		message, err := service.Save(args.ResumeContent, args.FileName)
		if err != nil {
			return errResult(err)
		}
		return textResult(message), nil
	}
}

func textResult(text string) CallToolResult {
	return CallToolResult{Content: []ContentBlock{{Type: "text", Text: text}}}
}

func errResult(err error) (CallToolResult, error) {
	return CallToolResult{}, err
}
