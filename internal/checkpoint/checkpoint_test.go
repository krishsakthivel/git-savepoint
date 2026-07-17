package checkpoint_test

import (
	"strings"
	"testing"

	"git-savepoint/internal/checkpoint"
	"git-savepoint/internal/gitutil"
	"git-savepoint/internal/storage"
	"git-savepoint/internal/testutil"
)

func TestCreateBasic(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")
	testutil.WriteFile(t, repo, "f.txt", "v2")

	cp, err := checkpoint.Create(repo, "my checkpoint")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cp.Message != "my checkpoint" {
		t.Errorf("message = %q, want %q", cp.Message, "my checkpoint")
	}

	all, err := storage.List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 checkpoint, got %d", len(all))
	}
}

func TestCreateNoChangesReturnsErrNoChanges(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	if _, err := checkpoint.Create(repo, "cp1"); err != nil {
		t.Fatalf("first checkpoint: %v", err)
	}

	if _, err := checkpoint.Create(repo, "cp2"); err != checkpoint.ErrNoChanges {
		t.Errorf("Create with no changes: err = %v, want ErrNoChanges", err)
	}
}

func TestCreateDoesNotDisturbRealStagingArea(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	testutil.WriteFile(t, repo, "staged.txt", "deliberately staged")
	if _, err := gitutil.Run(repo, "add", "staged.txt"); err != nil {
		t.Fatal(err)
	}

	testutil.WriteFile(t, repo, "unstaged.txt", "not staged")

	if _, err := checkpoint.Create(repo, "cp"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	status, err := gitutil.Run(repo, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, "A  staged.txt") {
		t.Errorf("checkpoint.Create disturbed the real staging area, git status:\n%s", status)
	}
	if !strings.Contains(status, "?? unstaged.txt") {
		t.Errorf("expected unstaged.txt to remain untracked, git status:\n%s", status)
	}
}

func TestCreateIgnoresNodeModulesAndGitDir(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	testutil.WriteFile(t, repo, "node_modules/pkg/index.js", "module.exports = {}")
	testutil.WriteFile(t, repo, "real.txt", "should be included")

	cp, err := checkpoint.Create(repo, "cp with ignores")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	out, err := gitutil.Run(repo, "ls-tree", "-r", "--name-only", cp.Commit)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "node_modules") {
		t.Errorf("checkpoint tree should not contain node_modules, got:\n%s", out)
	}
	if !strings.Contains(out, "real.txt") {
		t.Errorf("checkpoint tree should contain real.txt, got:\n%s", out)
	}
}

func TestCreateOnRepoWithNoCommitsYet(t *testing.T) {
	repo := testutil.NewRepo(t)

	testutil.WriteFile(t, repo, "f.txt", "v1")

	cp, err := checkpoint.Create(repo, "first ever checkpoint")
	if err != nil {
		t.Fatalf("Create on a repo with no commits: %v", err)
	}
	if cp.Commit == "" {
		t.Error("expected a non-empty commit hash")
	}
}

func TestCreateChecksAreIndependentCheckpoints(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	testutil.WriteFile(t, repo, "f.txt", "v2")
	cp1, err := checkpoint.Create(repo, "checkpoint 1")
	if err != nil {
		t.Fatalf("Create cp1: %v", err)
	}

	testutil.WriteFile(t, repo, "f.txt", "v3")
	cp2, err := checkpoint.Create(repo, "checkpoint 2")
	if err != nil {
		t.Fatalf("Create cp2: %v", err)
	}

	if cp1.Commit == cp2.Commit {
		t.Error("two checkpoints with different content produced the same commit hash")
	}

	all, err := storage.List(repo)
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 checkpoints, got %d (err=%v)", len(all), err)
	}
}
