package profiles

import (
	"fmt"
	"strings"

	genvalidator "janus/internal/core/generator/validator"
)

const (
	maxNameLen        = 100
	maxDescriptionLen = 500
)

// ValidateName checks profile name requirements:
// non-empty, no leading/trailing whitespace, max 100 characters.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("profile name must not have leading or trailing whitespace")
	}
	if len(name) > maxNameLen {
		return fmt.Errorf("profile name must not exceed %d characters (got %d)", maxNameLen, len(name))
	}
	return nil
}

// ValidateDescription checks description length.
// The field is optional; max 500 characters.
func ValidateDescription(desc string) error {
	if len(desc) > maxDescriptionLen {
		return fmt.Errorf("description must not exceed %d characters (got %d)", maxDescriptionLen, len(desc))
	}
	return nil
}

// ValidateProfile performs complete validation of a profile.
// Options are validated when they are populated (FileCount > 0 or TotalSize set).
func ValidateProfile(p *Profile) error {
	if p == nil {
		return fmt.Errorf("profile is nil")
	}
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if err := ValidateDescription(p.Description); err != nil {
		return err
	}

	// Only validate options when they carry meaningful data.
	// An empty options block is allowed for template-style profiles.
	if p.Options.FileCount > 0 || p.Options.TotalSize != "" {
		enhancedOpts, err := p.Options.ToEnhancedOptions()
		if err != nil {
			return fmt.Errorf("options: %w", err)
		}
		v := genvalidator.New()
		result := v.ValidateAll(enhancedOpts)
		if result.HasErrors() {
			return fmt.Errorf("options validation failed:\n%s", result.ErrorMessages())
		}
	}

	return nil
}
