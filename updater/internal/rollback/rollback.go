package rollback

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/observiq/observiq-otel-collector/updater/internal/file"
	"github.com/observiq/observiq-otel-collector/updater/internal/path"
	"github.com/observiq/observiq-otel-collector/updater/internal/service"
	"go.uber.org/multierr"
)

//go:generate mockery --name ActionAppender --filename action_appender.go
type ActionAppender interface {
	AppendAction(action RollbackableAction)
}

type Rollbacker struct {
	originalSvc service.Service
	backupDir   string
	latestDir   string
	installDir  string
	tmpDir      string
	actions     []RollbackableAction
}

func NewRollbacker(tmpDir string) (*Rollbacker, error) {
	installDir, err := path.InstallDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine install dir: %w", err)
	}

	return &Rollbacker{
		backupDir:  path.BackupDirFromTempDir(tmpDir),
		latestDir:  path.LatestDirFromTempDir(tmpDir),
		installDir: installDir,
		tmpDir:     tmpDir,
	}, nil
}

// AppendAction records the action that was performed, so that it may be undone later.
func (r *Rollbacker) AppendAction(action RollbackableAction) {
	r.actions = append(r.actions, action)
}

// Backup backs up the installDir to the rollbackDir
func (r Rollbacker) Backup() error {
	// Copy all the files in the install directory to the backup directory
	if err := copyFiles(r.installDir, r.backupDir, r.tmpDir); err != nil {
		return fmt.Errorf("failed to copy files to backup dir: %w", err)
	}

	// Backup the service configuration so we can reload it in case of rollback
	if err := r.originalSvc.Backup(path.ServiceFileDir(r.backupDir)); err != nil {
		return fmt.Errorf("failed to backup service configuration: %w", err)
	}

	return nil
}

// Rollback performs a rollback by undoing all recorded actions.
func (r Rollbacker) Rollback() (err error) {
	// We need to loop through the actions slice backwards, to roll back the actions in the correct order.
	// e.g. if StartService was called last, we need to stop the service first, then rollback previous actions.
	for i := len(r.actions) - 1; i >= 0; i-- {
		action := r.actions[i]
		// TODO: Evaluate if we need to use multierr here (I think it's useful, but others seem to hate this)
		// Maybe we even just log and don't return the error upstream.
		err = multierr.Append(err, action.Rollback())
	}
	return
}

// copyFiles moves the file tree rooted at latestDirPath to installDirPath,
// skipping configuration files. Appends CopyFileAction-s to the Rollbacker as it copies file.
func copyFiles(inputPath, outputPath, tmpDir string) error {
	absTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for temporary directory: %w", err)
	}

	err = filepath.WalkDir(inputPath, func(inPath string, d fs.DirEntry, err error) error {

		fullPath, absErr := filepath.Abs(inputPath)
		if absErr != nil {
			return fmt.Errorf("failed to determine absolute path of file: %w", err)
		}

		switch {
		case err != nil:
			// if there was an error walking the directory, we want to bail out.
			return err
		case d.IsDir() && strings.HasPrefix(fullPath, absTmpDir):
			// If this is the directory "tmp", we want to skip copying this directory,
			// since this folder is only for temporary files (and is where this binary is running right now)
			return filepath.SkipDir
		case d.IsDir():
			// Skip directories, we'll create them when we get a file in the directory.
			return nil
		}

		// We want the path relative to the directory we are walking in order to calculate where the file should be
		// mirrored in the output directory.
		relPath, err := filepath.Rel(inputPath, inPath)
		if err != nil {
			return err
		}

		// use the relative path to get the outPath (where we should write the file), and
		// to get the out directory (which we will create if it does not exist).
		outPath := filepath.Join(outputPath, relPath)
		outDir := filepath.Dir(outPath)

		if err := os.MkdirAll(outDir, 0750); err != nil {
			return fmt.Errorf("failed to create dir: %w", err)
		}

		if err := file.CopyFile(inPath, outPath); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk latest dir: %w", err)
	}

	return nil
}
