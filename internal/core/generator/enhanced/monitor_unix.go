//go:build unix || linux || darwin
// +build unix linux darwin

package enhanced

import (
	"fmt"
	"syscall"
	"time"

	"janus/internal/core/generator/models"
)

// CheckDiskSpace checks if we still have enough disk space (Unix/Linux/Mac implementation).
// Returns a non-empty warning string when space is low but not yet critical.
func (m *GenerationMonitor) CheckDiskSpace() (warning string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Only check periodically (expensive operation)
	if time.Since(m.LastDiskCheck) < m.DiskCheckInterval {
		return "", nil
	}

	// Get current disk stats
	var stat syscall.Statfs_t
	if statErr := syscall.Statfs(m.OutputPath, &stat); statErr != nil {
		return "", fmt.Errorf("disk check failed: %w", statErr)
	}

	freeSpace := int64(stat.Bavail) * int64(stat.Bsize)

	// EMERGENCY: Less than safety margin remaining
	if freeSpace < m.SafetyMargin {
		return "", fmt.Errorf("CRITICAL: Only %s free space remaining (need %s safety margin)",
			models.FormatBytes(freeSpace),
			models.FormatBytes(m.SafetyMargin))
	}

	// WARNING: Getting close (less than 2x safety margin)
	if freeSpace < m.SafetyMargin*2 {
		warning = fmt.Sprintf("Disk space getting low (%s remaining)", models.FormatBytes(freeSpace))
	}

	m.LastDiskCheck = time.Now()
	return warning, nil
}
