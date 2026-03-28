//go:build unix || linux || darwin
// +build unix linux darwin

package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"janus/internal/core/generator/models"
)

// checkDiskSpace checks available disk space for the given path (Unix/Linux/Mac implementation)
func (v *Validator) checkDiskSpace(path string, requiredSpace int64) (*models.DiskSpaceInfo, error) {
	// Ensure path exists or use parent
	checkPath := path
	if _, err := os.Stat(path); os.IsNotExist(err) {
		checkPath = filepath.Dir(path)
	}
	
	// Get disk stats using Unix syscall
	var stat syscall.Statfs_t
	if err := syscall.Statfs(checkPath, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}
	
	info := &models.DiskSpaceInfo{
		TotalSpace:    int64(stat.Blocks) * int64(stat.Bsize),
		FreeSpace:     int64(stat.Bavail) * int64(stat.Bsize),
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
