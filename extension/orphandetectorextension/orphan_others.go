// +build !windows

package orphandetectorextension

import (
	"os"

	"go.uber.org/zap"
)

func orphan(ppid int, dieIfInit bool, logger *zap.Logger) bool {
	osppid := os.Getppid()
	if ppid != osppid {
		logger.Info("Parent Process has exited.")
		logger.Debug("Parent Process ID Changed", zap.Int("orig_ppid", ppid), zap.Int("new_ppid", osppid))
		return true
	}

	if dieIfInit && osppid == 1 {
		logger.Info("Parent Process ID is 1")
		return true
	}

	return false
}
