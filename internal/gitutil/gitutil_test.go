package gitutil_test

import (
	"testing"

	"git-savepoint/internal/gitutil"
	"git-savepoint/internal/testutil"
)

func TestUnsupportedFeaturesCleanRepo(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	issues, err := gitutil.UnsupportedFeatures(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues on a clean repo, got %v", issues)
	}
}

func TestUnsupportedFeaturesSubmodule(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, ".gitmodules", "[submodule \"x\"]\n\tpath = x\n\turl = /tmp/foo\n")
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	issues, err := gitutil.UnsupportedFeatures(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(issues, "submodules") {
		t.Errorf("expected submodules to be flagged, got %v", issues)
	}
}

func TestUnsupportedFeaturesGitLFS(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, ".gitattributes", "*.psd filter=lfs diff=lfs merge=lfs -text\n")
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	issues, err := gitutil.UnsupportedFeatures(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(issues, "Git LFS") {
		t.Errorf("expected Git LFS to be flagged, got %v", issues)
	}
}

func TestUnsupportedFeaturesSparseCheckout(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	if _, err := gitutil.Run(repo, "sparse-checkout", "init"); err != nil {
		t.Fatalf("git sparse-checkout init: %v", err)
	}

	issues, err := gitutil.UnsupportedFeatures(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(issues, "sparse checkout") {
		t.Errorf("expected sparse checkout to be flagged, got %v", issues)
	}
}

func TestUnsupportedFeaturesMergeInProgress(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "a")
	testutil.Commit(t, repo, "commit 1")

	gitutil.Run(repo, "checkout", "-q", "-b", "feature")
	testutil.WriteFile(t, repo, "f.txt", "a\nb")
	testutil.Commit(t, repo, "feature change")

	gitutil.Run(repo, "checkout", "-q", "main")
	testutil.WriteFile(t, repo, "f.txt", "a\nc")
	testutil.Commit(t, repo, "main change")

	gitutil.Run(repo, "merge", "feature")

	issues, err := gitutil.UnsupportedFeatures(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(issues, "a merge in progress") {
		t.Errorf("expected merge-in-progress to be flagged, got %v", issues)
	}
}

func TestUnsupportedFeaturesDetachedHeadIsFine(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "commit 1")
	testutil.WriteFile(t, repo, "f.txt", "v2")
	testutil.Commit(t, repo, "commit 2")

	if _, err := gitutil.Run(repo, "checkout", "-q", "HEAD~1"); err != nil {
		t.Fatalf("checkout detached: %v", err)
	}

	issues, err := gitutil.UnsupportedFeatures(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Errorf("a detached HEAD should not be flagged as unsupported, got %v", issues)
	}
}

func containsStr(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
