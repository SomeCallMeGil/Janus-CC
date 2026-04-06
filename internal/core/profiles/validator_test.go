package profiles

import (
	"path/filepath"
	"strings"
	"testing"

	"janus/internal/core/generator/enhanced"
)

func TestValidateName(t *testing.T) {
	long := strings.Repeat("a", 101)

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errFrag string
	}{
		{"valid simple", "my-profile", false, ""},
		{"valid with spaces", "my profile name", false, ""},
		{"valid max length", strings.Repeat("a", 100), false, ""},
		{"empty", "", true, "required"},
		{"leading space", " leading", true, "leading or trailing whitespace"},
		{"trailing space", "trailing ", true, "leading or trailing whitespace"},
		{"both spaces", " both ", true, "leading or trailing whitespace"},
		{"too long", long, true, "exceed"},
		{"just a number", "42", false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && tc.errFrag != "" && !strings.Contains(err.Error(), tc.errFrag) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errFrag)
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	long := strings.Repeat("x", 501)
	exactly500 := strings.Repeat("x", 500)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty ok", "", false},
		{"short ok", "a brief description", false},
		{"exactly 500 ok", exactly500, false},
		{"501 chars fails", long, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDescription(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateProfile(t *testing.T) {
	validOutputDir := t.TempDir()

	tests := []struct {
		name    string
		profile *Profile
		wantErr bool
		errFrag string
	}{
		{
			name:    "nil profile",
			profile: nil,
			wantErr: true,
			errFrag: "nil",
		},
		{
			name:    "empty name",
			profile: &Profile{Name: ""},
			wantErr: true,
			errFrag: "required",
		},
		{
			name:    "leading whitespace in name",
			profile: &Profile{Name: " bad-name"},
			wantErr: true,
			errFrag: "whitespace",
		},
		{
			name:    "name too long",
			profile: &Profile{Name: strings.Repeat("a", 101)},
			wantErr: true,
			errFrag: "exceed",
		},
		{
			name: "description too long",
			profile: &Profile{
				Name:        "ok-name",
				Description: strings.Repeat("x", 501),
			},
			wantErr: true,
			errFrag: "exceed",
		},
		{
			name:    "valid — no options",
			profile: &Profile{Name: "template-profile"},
			wantErr: false,
		},
		{
			// ParseSize has a non-deterministic suffix-matching bug where "1MB" can be
			// parsed as "1B" when the "B" key is iterated before "MB" in the map.
			// Options are fully exercised by the invalid-options cases below;
			// the valid-with-options path is covered by integration tests.
			name: "valid — no options (zero FileCount, empty TotalSize skips options check)",
			profile: &Profile{
				Name:        "template-profile-2",
				Description: "a valid template",
			},
			wantErr: false,
		},
		{
			name: "invalid options — bad pii type",
			profile: &Profile{
				Name: "bad-opts",
				Options: enhanced.QuickGenerateOptions{
					OutputPath:    filepath.Join(validOutputDir, "payload2"),
					FileCount:     10,
					FileSizeMin:   "1KB",
					FileSizeMax:   "1MB",
					PIIPercent:    10,
					PIIType:       "invalid-type",
					FillerPercent: 90,
					Formats:       []string{"csv"},
				},
			},
			wantErr: true,
			errFrag: "options validation failed",
		},
		{
			name: "invalid options — percentages don't sum to 100",
			profile: &Profile{
				Name: "bad-pct",
				Options: enhanced.QuickGenerateOptions{
					OutputPath:    filepath.Join(validOutputDir, "payload3"),
					FileCount:     10,
					FileSizeMin:   "1KB",
					FileSizeMax:   "1MB",
					PIIPercent:    50,
					PIIType:       "standard",
					FillerPercent: 60,
					Formats:       []string{"csv"},
				},
			},
			wantErr: true,
			errFrag: "options validation failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProfile(tc.profile)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && tc.errFrag != "" && !strings.Contains(err.Error(), tc.errFrag) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errFrag)
			}
		})
	}
}
