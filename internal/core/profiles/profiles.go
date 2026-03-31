// Package profiles provides named, reusable generation configurations.
package profiles

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"janus/internal/core/generator/enhanced"
)

// Profile is a named, stored set of QuickGenerateOptions.
type Profile struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Options     enhanced.QuickGenerateOptions `json:"options"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
}

// Store is the persistence interface for profiles.
type Store interface {
	Create(p *Profile) error
	Get(id string) (*Profile, error)
	GetByName(name string) (*Profile, error)
	List() ([]*Profile, error)
	Update(p *Profile) error
	Delete(id string) error
}

// Manager handles profile operations.
type Manager struct {
	store Store
}

// New creates a new profile manager.
func New(store Store) *Manager {
	return &Manager{store: store}
}

// Create creates and persists a new profile.
func (m *Manager) Create(name, description string, opts enhanced.QuickGenerateOptions) (*Profile, error) {
	if name == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	// Check for name collision
	if existing, _ := m.store.GetByName(name); existing != nil {
		return nil, fmt.Errorf("profile %q already exists", name)
	}

	opts.Name = name
	p := &Profile{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Options:     opts,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.store.Create(p); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}
	return p, nil
}

// Get retrieves a profile by ID.
func (m *Manager) Get(id string) (*Profile, error) {
	return m.store.Get(id)
}

// GetByName retrieves a profile by name.
func (m *Manager) GetByName(name string) (*Profile, error) {
	p, err := m.store.GetByName(name)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// List returns all profiles.
func (m *Manager) List() ([]*Profile, error) {
	return m.store.List()
}

// Update applies a partial update to a profile.
func (m *Manager) Update(id string, updates map[string]interface{}) (*Profile, error) {
	p, err := m.store.Get(id)
	if err != nil {
		return nil, err
	}

	if name, ok := updates["name"].(string); ok && name != "" {
		// Check name collision with other profiles
		if existing, _ := m.store.GetByName(name); existing != nil && existing.ID != id {
			return nil, fmt.Errorf("profile %q already exists", name)
		}
		p.Name = name
		p.Options.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		p.Description = desc
	}
	if optsRaw, ok := updates["options"]; ok {
		// Re-marshal to JSON then back into QuickGenerateOptions for clean merge
		b, err := json.Marshal(optsRaw)
		if err != nil {
			return nil, fmt.Errorf("marshal options: %w", err)
		}
		if err := json.Unmarshal(b, &p.Options); err != nil {
			return nil, fmt.Errorf("unmarshal options: %w", err)
		}
	}

	p.UpdatedAt = time.Now()

	if err := m.store.Update(p); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return p, nil
}

// Delete removes a profile by ID.
func (m *Manager) Delete(id string) error {
	return m.store.Delete(id)
}

// Resolve returns the QuickGenerateOptions for a profile, optionally
// merging caller-supplied overrides on top of the saved values.
func (m *Manager) Resolve(id string, overrides map[string]interface{}) (enhanced.QuickGenerateOptions, error) {
	p, err := m.store.Get(id)
	if err != nil {
		return enhanced.QuickGenerateOptions{}, err
	}

	opts := p.Options

	if overrides == nil {
		return opts, nil
	}

	// Merge overrides — re-encode the profile options, overlay the overrides, decode back
	base, err := json.Marshal(opts)
	if err != nil {
		return opts, fmt.Errorf("marshal base options: %w", err)
	}

	var merged map[string]interface{}
	if err := json.Unmarshal(base, &merged); err != nil {
		return opts, fmt.Errorf("unmarshal base options: %w", err)
	}
	for k, v := range overrides {
		merged[k] = v
	}

	merged2, err := json.Marshal(merged)
	if err != nil {
		return opts, fmt.Errorf("re-marshal merged options: %w", err)
	}
	if err := json.Unmarshal(merged2, &opts); err != nil {
		return opts, fmt.Errorf("unmarshal merged options: %w", err)
	}

	return opts, nil
}

// DefaultProfiles returns a set of pre-built starter profiles.
var DefaultProfiles = []struct {
	Name        string
	Description string
	Options     enhanced.QuickGenerateOptions
}{
	{
		Name:        "quick-pii-test",
		Description: "1000 standard PII files — fast smoke test",
		Options: enhanced.QuickGenerateOptions{
			OutputPath:     "./payloads/quick-pii",
			FileCount:      1000,
			FileSizeMin:    "1KB",
			FileSizeMax:    "1MB",
			PIIPercent:     100,
			PIIType:        "standard",
			FillerPercent:  0,
			Formats:        []string{"csv", "json", "txt"},
			DirectoryDepth: 3,
		},
	},
	{
		Name:        "mixed-realistic",
		Description: "1 GB mixed dataset — 15% PII, 85% filler",
		Options: enhanced.QuickGenerateOptions{
			OutputPath:     "./payloads/mixed",
			TotalSize:      "1GB",
			FileSizeMin:    "1KB",
			FileSizeMax:    "10MB",
			PIIPercent:     15,
			PIIType:        "standard",
			FillerPercent:  85,
			Formats:        []string{"csv", "json", "txt"},
			DirectoryDepth: 3,
		},
	},
	{
		Name:        "healthcare-large",
		Description: "5 GB healthcare dataset — 30% medical PII, 70% filler",
		Options: enhanced.QuickGenerateOptions{
			OutputPath:     "./payloads/healthcare",
			TotalSize:      "5GB",
			FileSizeMin:    "10KB",
			FileSizeMax:    "50MB",
			PIIPercent:     30,
			PIIType:        "healthcare",
			FillerPercent:  70,
			Formats:        []string{"csv", "json"},
			DirectoryDepth: 4,
		},
	},
	{
		Name:        "financial-audit",
		Description: "5000 financial records — 40% financial PII, 60% filler",
		Options: enhanced.QuickGenerateOptions{
			OutputPath:     "./payloads/financial",
			FileCount:      5000,
			FileSizeMin:    "5KB",
			FileSizeMax:    "5MB",
			PIIPercent:     40,
			PIIType:        "financial",
			FillerPercent:  60,
			Formats:        []string{"csv", "json", "txt"},
			DirectoryDepth: 3,
		},
	},
	{
		Name:        "compliance-10pct",
		Description: "10 GB compliance test — 10% PII simulating a typical enterprise dataset",
		Options: enhanced.QuickGenerateOptions{
			OutputPath:     "./payloads/compliance",
			TotalSize:      "10GB",
			FileSizeMin:    "1MB",
			FileSizeMax:    "100MB",
			PIIPercent:     10,
			PIIType:        "standard",
			FillerPercent:  90,
			Formats:        []string{"csv", "json", "txt"},
			DirectoryDepth: 4,
		},
	},
}
