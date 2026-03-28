//go:build windows
// +build windows

package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"janus/internal/core/generator/models"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// checkDiskSpace checks available disk space for the given path (Windows implementation)
func (v *Validator) checkDiskSpace(path string, requiredSpace int64) (*models.DiskSpaceInfo, error) {
	// Ensure path exists or use parent
	checkPath := path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		checkPath = filepath.Dir(path)
	}
	
	// Convert to absolute path
	absPath, err := filepath.Abs(checkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// Get the drive root (e.g., C:\)
	vol := filepath.VolumeName(absPath)
	if vol == "" {
		vol = absPath
	}
	
	// Call Windows API to get disk space
	var freeBytesAvailable, totalBytes, totalFreeBytes int64
	
	volumePtr, err := syscall.UTF16PtrFromString(vol + "\\")
	if err != nil {
		return nil, fmt.Errorf("failed to convert path: %w", err)
	}
	
	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(volumePtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	
	if ret == 0 {
		return nil, fmt.Errorf("failed to get disk space: %w", err)
	}
	
	info := &models.DiskSpaceInfo{
		TotalSpace:    totalBytes,
		FreeSpace:     freeBytesAvailable,
		RequiredSpace: requiredSpace,
	}
	
	info.UsedSpace = info.TotalSpace - info.FreeSpace
	info.UsagePercent = float64(info.UsedSpace) / float64(info.TotalSpace) * 100
	
	// Calculate safety margin (25% of total OR 5GB, whichever is larger)
	percentMargin := int64(float64(info.TotalSpace) * (MinimumSafetyMarginPercent / 100.0))
	if percentMargin < AbsoluteSafetyMargin {
		info.SafetyMargin = AbsoluteSafetyMargin
	} else {
		info.SafetyMargin = percentMargin
	}
	
	info.Available = info.FreeSpace - info.SafetyMargin
	info.Sufficient = info.Available >= requiredSpace
	info.AfterGenPercent = float64(info.UsedSpace+requiredSpace) / float64(info.TotalSpace) * 100
	
	return info, nil
}
