package jobs

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CreatePayloadDirectory creates the standard directory structure for a job run.
// Returns the root payload path: {baseDir}/payloads/{runID}/
func CreatePayloadDirectory(runID, baseDir string) (string, error) {
	payloadPath := filepath.Join(baseDir, "payloads", runID)

	dirs := []string{
		payloadPath,
		filepath.Join(payloadPath, "output"),
		filepath.Join(payloadPath, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
	}

	return payloadPath, nil
}

// PayloadMetadata is the schema written to metadata.json inside the payload directory.
type PayloadMetadata struct {
	RunID      string          `json:"run_id"`
	ProfileID  string          `json:"profile_id,omitempty"`
	Config     json.RawMessage `json:"config,omitempty"`
	StartTime  string          `json:"start_time"`
	OutputPath string          `json:"output_path"`
}

// SavePayloadMetadata writes config.json, an optional profile.json, and metadata.json
// into payloadPath. profileJSON may be nil if no profile is associated with the run.
func SavePayloadMetadata(payloadPath string, job *JobRun, profileJSON json.RawMessage) error {
	// config.json — snapshot of generation options
	if len(job.Config) > 0 {
		if err := writeJSON(filepath.Join(payloadPath, "config.json"), job.Config); err != nil {
			return err
		}
	}

	// profile.json — snapshot of the profile (if run was triggered by one)
	if len(profileJSON) > 0 {
		if err := writeJSON(filepath.Join(payloadPath, "profile.json"), profileJSON); err != nil {
			return err
		}
	}

	// metadata.json — top-level summary
	meta := PayloadMetadata{
		RunID:      job.ID,
		ProfileID:  job.ProfileID,
		Config:     job.Config,
		StartTime:  job.StartTime.UTC().Format("2006-01-02T15:04:05Z"),
		OutputPath: job.OutputPath,
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(payloadPath, "metadata.json"), b)
}

func writeJSON(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
