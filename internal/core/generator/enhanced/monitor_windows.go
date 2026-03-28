//go:build windows
// +build windows

package enhanced

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"janus/internal/core/generator/models"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// CheckDiskSpace checks if we still have enough disk space (Windows implementation)
func (m *GenerationMonitor) CheckDiskSpace() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Only check periodically (expensive operation)
	if time.Since(m.LastDiskCheck) < m.DiskCheckInterval {
		return nil
	}
	
	// Get the drive root (e.g., C:\)
	absPath, err := filepath.Abs(m.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	vol := filepath.VolumeName(absPath)
	if vol == "" {
		vol = absPath
	}
	
	// Call Windows API to get disk space
	var freeBytesAvailable, totalBytes, totalFreeBytes int64
	
	volumePtr, err := syscall.UTF16PtrFromString(vol + "\\")
	if err != nil {
		return fmt.Errorf("failed to convert path: %w", err)
	}
	
	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(volumePtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	
	if ret == 0 {
		return fmt.Errorf("disk check failed: %w", err)
	}
	
	freeSpace := freeBytesAvailable
	
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
