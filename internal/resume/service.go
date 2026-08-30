package resume

import (
	"fmt"
	"strings"
)

type DataSource interface {
	ReadProfile() ([]byte, error)
	ReadResumeFormat() ([]byte, error)
}

const maxResumeContentSize = 2 << 20 // 2 MiB

type Service struct {
	data   DataSource
	writer Writer
}

func NewService(data DataSource, writer Writer) Service {
	return Service{data: data, writer: writer}
}

func (s Service) PrepareJobContext(jobDescription string) (string, error) {
	if strings.TrimSpace(jobDescription) == "" {
		return "", fmt.Errorf("the 'job_description' field cannot be empty")
	}
	if len(jobDescription) > maxResumeContentSize {
		return "", fmt.Errorf("the 'job_description' field is too long")
	}

	profile, err := s.data.ReadProfile()
	if err != nil {
		return "", err
	}
	format, err := s.data.ReadResumeFormat()
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("# Context for generating a tailored resume\n\n")
	builder.WriteString("## Instructions\n")
	builder.WriteString("First determine the predominant language of the profile's narrative content. Compare the prose in the summary, descriptions, achievements, education, certifications, and soft skills; do not count only names, company names, technologies, or URLs. Write the entire resume, including section headings and the fit evaluation, in the predominant language: Spanish when Spanish content is the majority, or English when English content is the majority. Then write a Markdown resume tailored to the job description below, using ")
	builder.WriteString("only factual information from the profile (do not invent experience, ")
	builder.WriteString("degrees, or dates). Prioritize and reorder the experience, achievements, and skills ")
	builder.WriteString("that are most relevant to this specific position. Follow the structure defined ")
	builder.WriteString("in 'resume_format'. The final fit evaluation section is mandatory and must include ")
	builder.WriteString("an integer score from 1 to 10, strengths, and weaknesses or gaps, ")
	builder.WriteString("clearly distinguishing demonstrated experience from unverified experience. When finished, ")
	builder.WriteString("save the result using the 'save_resume' tool.\n\n")

	builder.WriteString("## Job description\n```\n")
	builder.WriteString(jobDescription)
	builder.WriteString("\n```\n\n")
	builder.WriteString("## Candidate profile (data/profile.json)\n```json\n")
	builder.Write(profile)
	builder.WriteString("\n```\n\n")
	builder.WriteString("## Required resume format (resume_format.json)\n```json\n")
	builder.Write(format)
	builder.WriteString("\n```\n")

	return builder.String(), nil
}

func (s Service) Save(content, fileName string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("the 'content' field cannot be empty")
	}
	if len(content) > maxResumeContentSize {
		return "", fmt.Errorf("the 'content' field is too long")
	}
	return s.writer.Save(content, fileName)
}
