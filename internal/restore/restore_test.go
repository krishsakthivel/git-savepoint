package restore_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-savepoint/internal/checkpoint"
	"git-savepoint/internal/restore"
	"git-savepoint/internal/testutil"
)

func TestRestoreBasic(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	cp1, err := checkpoint.Create(repo, "checkpoint at v1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	testutil.WriteFile(t, repo, "f.txt", "v2 changed after checkpoint")

	result, err := restore.To(repo, "latest")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if result.RestoredTo.Commit != cp1.Commit {
		t.Errorf("restored to commit %s, want %s", result.RestoredTo.Commit, cp1.Commit)
	}

	content, err := os.ReadFile(filepath.Join(repo, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "v1" {
		t.Errorf("f.txt after restore = %q, want %q", content, "v1")
	}
}

func TestRestoreTakesSafetyCheckpointFirst(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	if _, err := checkpoint.Create(repo, "cp1"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	testutil.WriteFile(t, repo, "f.txt", "v2 uncommitted work")

	result, err := restore.To(repo, "latest")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if result.SafetyCheckpoint == nil {
		t.Fatal("expected a safety checkpoint to be taken before restoring")
	}

	safetyID := strings.TrimPrefix(result.SafetyCheckpoint.RefName(), "refs/git-savepoint/checkpoints/")
	if _, err := restore.To(repo, safetyID); err != nil {
		t.Fatalf("restoring to the safety checkpoint: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(repo, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "v2 uncommitted work" {
		t.Errorf("safety checkpoint didn't preserve pre-restore state, f.txt = %q", content)
	}
}

func TestRestoreSkipsSafetyCheckpointWhenNothingChanged(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	if _, err := checkpoint.Create(repo, "cp1"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	result, err := restore.To(repo, "latest")
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if result.SafetyCheckpoint != nil {
		t.Error("expected no safety checkpoint when working tree already matched the target")
	}
}

func TestRestoreUnknownID(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")
	if _, err := checkpoint.Create(repo, "cp1"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := restore.To(repo, "not-a-real-checkpoint-id"); err == nil {
		t.Error("expected an error for an unknown checkpoint id")
	}
}

func TestRestoreNoCheckpointsExist(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	if _, err := restore.To(repo, "latest"); err == nil {
		t.Error("expected an error when no checkpoints exist to restore to")
	}
}
