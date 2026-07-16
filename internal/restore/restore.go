package restore
 
import (
	"fmt"
 
	"git-savepoint/internal/checkpoint"
	"git-savepoint/internal/gitutil"
	"git-savepoint/internal/storage"
)
 

type Result struct {
	SafetyCheckpoint *storage.Checkpoint // nothing if working tree was already clean
	RestoredTo       storage.Checkpoint
}
 

func To(repoRoot string, id string) (Result, error) {
	target, err := storage.Find(repoRoot, id)
	if err != nil {
		return Result{}, err
	}
 
	var result Result
 
	safety, err := checkpoint.Create(repoRoot, fmt.Sprintf("Safety checkpoint before restoring to %s", target.Time().Format("15:04:05")))
	switch {
	case err == nil:
		result.SafetyCheckpoint = &safety
	case err == checkpoint.ErrNoChanges:
		// working tree already matches the last checkpoint
	default:
		return Result{}, fmt.Errorf("failed to take safety checkpoint, aborting restore: %w", err)
	}
 
	// hard-restore every tracked path to match the target commit's tree.
	if _, err := gitutil.Run(repoRoot, "checkout", target.Commit, "--", "."); err != nil {
		return Result{}, fmt.Errorf("restoring files: %w", err)
	}
 
	result.RestoredTo = target
	return result, nil
}