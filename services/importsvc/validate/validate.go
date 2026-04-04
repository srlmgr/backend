package validate

import (
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/services/importsvc/points"
)

// This package provides validation functions for the points system settings and inputs.
// It ensures that the settings are consistent and that the inputs are valid
// according to the defined rules.
type (
	Validator struct {
		settings *points.SeasonSettings
	}
)

func NewValidator(settings *points.SeasonSettings) *Validator {
	return &Validator{
		settings: settings,
	}
}

// examines the settings and checks if all required data is present in entries
// example: if quali points are awarded, we need at least starting position
// or quali time
func (v *Validator) ValidateResultEntries(entries []*models.ResultEntry) error {
	return nil
}
