package main

import mcpserver "resume-mcp/mcp"

func registerTools(srv *mcpserver.Server, app application) {
	srv.RegisterTool(mcpserver.Tool{
		Name:        "get_profile",
		Description: "Returns the user's personal details, experience, education, and skills as JSON. Use this first to review the available information before tailoring a resume.",
		InputSchema: mcpserver.InputSchema{Type: "object"},
	}, app.getProfileTool)

	srv.RegisterTool(mcpserver.Tool{
		Name:        "get_resume_format",
		Description: "Returns the structure and format that the generated resume must follow (sections, order, style, and length). Use it together with get_profile.",
		InputSchema: mcpserver.InputSchema{Type: "object"},
	}, app.getResumeFormatTool)

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
	}, app.prepareJobContextTool)

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
	}, app.saveResumeTool)
}
