package profile

import (
	"fmt"
	"os"
)

type Repository struct {
	profilePath      string
	resumeFormatPath string
}

func NewRepository(profilePath, resumeFormatPath string) Repository {
	return Repository{
		profilePath:      profilePath,
		resumeFormatPath: resumeFormatPath,
	}
}

func (r Repository) ReadProfile() ([]byte, error) {
	data, err := os.ReadFile(r.profilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", r.profilePath, err)
	}
	return data, nil
}

func (r Repository) ReadResumeFormat() ([]byte, error) {
	data, err := os.ReadFile(r.resumeFormatPath)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", r.resumeFormatPath, err)
	}
	return data, nil
}
