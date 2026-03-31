package profiles

import (
	"encoding/json"
	"fmt"

	"janus/internal/core/generator/enhanced"
	"janus/internal/database/sqlite"
)

// SQLiteStore implements Store using the SQLite database.
type SQLiteStore struct {
	db *sqlite.SQLiteDB
}

// NewSQLiteStore creates a SQLiteStore.
func NewSQLiteStore(db *sqlite.SQLiteDB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Create(p *Profile) error {
	b, err := json.Marshal(p.Options)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	return s.db.CreateProfile(p.ID, p.Name, p.Description, string(b), p.CreatedAt, p.UpdatedAt)
}

func (s *SQLiteStore) Get(id string) (*Profile, error) {
	pid, name, desc, optJSON, createdAt, updatedAt, err := s.db.GetProfile(id)
	if err != nil {
		return nil, err
	}
	var opts enhanced.QuickGenerateOptions
	if err := json.Unmarshal([]byte(optJSON), &opts); err != nil {
		return nil, fmt.Errorf("unmarshal options: %w", err)
	}
	return &Profile{
		ID:          pid,
		Name:        name,
		Description: desc,
		Options:     opts,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *SQLiteStore) GetByName(name string) (*Profile, error) {
	id, pname, desc, optJSON, createdAt, updatedAt, err := s.db.GetProfileByName(name)
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, nil // not found
	}
	var opts enhanced.QuickGenerateOptions
	if err := json.Unmarshal([]byte(optJSON), &opts); err != nil {
		return nil, fmt.Errorf("unmarshal options: %w", err)
	}
	return &Profile{
		ID:          id,
		Name:        pname,
		Description: desc,
		Options:     opts,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *SQLiteStore) List() ([]*Profile, error) {
	rows, err := s.db.ListProfiles()
	if err != nil {
		return nil, err
	}
	result := make([]*Profile, 0, len(rows))
	for _, r := range rows {
		var opts enhanced.QuickGenerateOptions
		if err := json.Unmarshal([]byte(r.OptionsJSON), &opts); err != nil {
			return nil, fmt.Errorf("unmarshal options for %s: %w", r.ID, err)
		}
		result = append(result, &Profile{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Options:     opts,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	return result, nil
}

func (s *SQLiteStore) Update(p *Profile) error {
	b, err := json.Marshal(p.Options)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	return s.db.UpdateProfile(p.ID, p.Name, p.Description, string(b), p.UpdatedAt)
}

func (s *SQLiteStore) Delete(id string) error {
	return s.db.DeleteProfile(id)
}
