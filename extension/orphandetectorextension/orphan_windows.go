// +build windows

package orphandetectorextension

import (
	"os"

	"go.uber.org/zap"
)

func orphan(ppid int, dieIfInit bool, logger *zap.Logger) bool {
	process, err := os.FindProcess(ppid)
	defer process.Release()

	if err != nil || process == nil {
		logger.Info("Could not find Parent Process (%d): %w", zap.Int("ppid", ppid), zap.Error(err))
		return true
	}

	return false
}
