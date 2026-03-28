//go:build unix || linux || darwin
// +build unix linux darwin

package enhanced

import (
	"fmt"
	"syscall"
	"time"

	"janus/internal/core/generator/models"
)

// CheckDiskSpace checks if we still have enough disk space (Unix/Linux/Mac implementation)
func (m *GenerationMonitor) CheckDiskSpace() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Only check periodically (expensive operation)
	if time.Since(m.LastDiskCheck) < m.DiskCheckInterval {
		return nil
	}
	
	// Get current disk stats
	var stat syscall.Statfs_t
	if err := syscall.Statfs(m.OutputPath, &stat); err != nil {
		return fmt.Errorf("disk check failed: %w", err)
	}
	
	freeSpace := int64(stat.Bavail) * int64(stat.Bsize)
	
	// EMERGENCY: Less than safety margin remaining
	if freeSpace < m.SafetyMargin {
		return fmt.Errorf("CRITICAL: Only %s free space remaining (need %s safety margin)",
			models.FormatBytes(freeSpace),
			models.FormatBytes(m.SafetyMargin))
	}
	
	// WARNING: Getting close (less than 2x safety margin)
	if freeSpace < m.SafetyMargin*2 {
		fmt.Printf("⚠️ WARNING: Disk space getting low (%s remaining)\n", models.FormatBytes(freeSpace))
	}
	
	m.LastDiskCheck = time.Now()
	return nil
}
