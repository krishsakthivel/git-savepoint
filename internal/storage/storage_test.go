package storage_test

import (
	"fmt"
	"testing"

	"git-savepoint/internal/gitutil"
	"git-savepoint/internal/storage"
	"git-savepoint/internal/testutil"
)

func TestSaveAndList(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "hello")
	testutil.Commit(t, repo, "init")
	head := gitutil.HeadCommit(repo)

	cp, err := storage.Save(repo, head, "test checkpoint")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if cp.Commit != head {
		t.Errorf("commit = %s, want %s", cp.Commit, head)
	}
	if cp.Message != "test checkpoint" {
		t.Errorf("message = %q, want %q", cp.Message, "test checkpoint")
	}

	all, err := storage.List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("List returned %d checkpoints, want 1", len(all))
	}
	if all[0].Commit != head {
		t.Errorf("listed commit = %s, want %s", all[0].Commit, head)
	}
}

func TestListEmptyRepo(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "hello")
	testutil.Commit(t, repo, "init")

	all, err := storage.List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected no checkpoints, got %d", len(all))
	}
}

func TestListIsSortedOldestFirst(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")
	head := gitutil.HeadCommit(repo)

	for _, ts := range []int64{500, 100, 300} {
		ref := storage.RefPrefix + fmt.Sprint(ts)
		if _, err := gitutil.Run(repo, "update-ref", ref, head); err != nil {
			t.Fatalf("update-ref: %v", err)
		}
	}

	all, err := storage.List(repo)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d checkpoints, want 3", len(all))
	}
	want := []int64{100, 300, 500}
	for i, cp := range all {
		if cp.Timestamp != want[i] {
			t.Errorf("position %d: timestamp = %d, want %d (full order: %v)", i, cp.Timestamp, want[i], all)
		}
	}
}

func TestLatest(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")
	head := gitutil.HeadCommit(repo)

	if _, ok, err := storage.Latest(repo); err != nil || ok {
		t.Fatalf("Latest on empty checkpoint list: ok=%v err=%v, want ok=false", ok, err)
	}

	for _, ts := range []int64{100, 300} {
		ref := storage.RefPrefix + fmt.Sprint(ts)
		if _, err := gitutil.Run(repo, "update-ref", ref, head); err != nil {
			t.Fatal(err)
		}
	}

	latest, ok, err := storage.Latest(repo)
	if err != nil || !ok {
		t.Fatalf("Latest: ok=%v err=%v", ok, err)
	}
	if latest.Timestamp != 300 {
		t.Errorf("Latest timestamp = %d, want 300", latest.Timestamp)
	}
}

func TestFindByTimestampLatestAndHashPrefix(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")
	head := gitutil.HeadCommit(repo)

	cp, err := storage.Save(repo, head, "cp1")
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	byTS, err := storage.Find(repo, fmt.Sprint(cp.Timestamp))
	if err != nil {
		t.Fatalf("Find by timestamp: %v", err)
	}
	if byTS.Commit != head {
		t.Errorf("Find by timestamp returned wrong commit")
	}

	byLatest, err := storage.Find(repo, "latest")
	if err != nil || byLatest.Commit != head {
		t.Fatalf("Find(latest): commit=%s err=%v", byLatest.Commit, err)
	}

	byHash, err := storage.Find(repo, head[:8])
	if err != nil || byHash.Commit != head {
		t.Fatalf("Find by hash prefix: commit=%s err=%v", byHash.Commit, err)
	}
}

func TestFindUnknownID(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")
	head := gitutil.HeadCommit(repo)
	if _, err := storage.Save(repo, head, "cp1"); err != nil {
		t.Fatal(err)
	}

	if _, err := storage.Find(repo, "totally-not-a-real-id"); err == nil {
		t.Error("expected an error for an unknown checkpoint id")
	}
}

func TestFindNoCheckpointsExist(t *testing.T) {
	repo := testutil.NewRepo(t)
	testutil.WriteFile(t, repo, "f.txt", "v1")
	testutil.Commit(t, repo, "init")

	if _, err := storage.Find(repo, "latest"); err == nil {
		t.Error("expected an error when no checkpoints exist yet")
	}
}
