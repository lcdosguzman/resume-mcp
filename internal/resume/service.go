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
	profileLanguage := predominantProfileLanguage(profile)
	builder.WriteString(fmt.Sprintf("Default language for the profile: %s. Use this as the default output language unless the user explicitly asks for another language. The resume must be written in the profile's predominant language, not the job description's language. ", profileLanguage))
	builder.WriteString("Determine the predominant language by comparing the prose in the summary, descriptions, achievements, education, certifications, and soft skills; do not count only names, company names, technologies, or URLs. Then write a Markdown resume tailored to the job description below, using only factual information from the profile (do not invent experience, ")
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

func predominantProfileLanguage(profileData []byte) string {
	text := strings.ToLower(string(profileData))
	spanishCount := strings.Count(text, "español") + strings.Count(text, "spanish") + strings.Count(text, "soy") + strings.Count(text, "ingrese") + strings.Count(text, "trabajo") + strings.Count(text, "desarrollo") + strings.Count(text, "cliente") + strings.Count(text, "equipo") + strings.Count(text, "sistemas") + strings.Count(text, "arquitectura")
	englishCount := strings.Count(text, "english") + strings.Count(text, "experience") + strings.Count(text, "team") + strings.Count(text, "software") + strings.Count(text, "engineer") + strings.Count(text, "developer") + strings.Count(text, "project") + strings.Count(text, "design") + strings.Count(text, "lead") + strings.Count(text, "data")
	if spanishCount >= englishCount {
		return "Spanish"
	}
	return "English"
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
