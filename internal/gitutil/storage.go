
package storage

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"git-savepoint/internal/gitutil"
)

const RefPrefix = "refs/git-savepoint/checkpoints/"


type Checkpoint struct {
	Timestamp int64  // unix seconds, also the ref name
	Commit    string // commit-tree hash this ref points to
	Message   string // short human-readable label
}

func (c Checkpoint) Time() time.Time {
	return time.Unix(c.Timestamp, 0)
}

func (c Checkpoint) RefName() string {
	return RefPrefix + strconv.FormatInt(c.Timestamp, 10)
}


func Save(repoRoot string, commitHash string, message string) (Checkpoint, error) {
	ts := time.Now().Unix()
	cp := Checkpoint{Timestamp: ts, Commit: commitHash, Message: message}
	_, err := gitutil.Run(repoRoot, "update-ref", cp.RefName(), commitHash)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("saving checkpoint ref: %w", err)
	}
	return cp, nil
}


func List(repoRoot string) ([]Checkpoint, error) {
	out, err := gitutil.Run(repoRoot, "for-each-ref",
		"--format=%(refname)%09%(objectname)",
		RefPrefix)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var checkpoints []Checkpoint
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		refname, hash := parts[0], parts[1]
		tsStr := strings.TrimPrefix(refname, RefPrefix)
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue
		}
		msg, _ := gitutil.Run(repoRoot, "log", "-1", "--format=%s", hash)
		checkpoints = append(checkpoints, Checkpoint{
			Timestamp: ts,
			Commit:    hash,
			Message:   msg,
		})
	}

	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp < checkpoints[j].Timestamp
	})
	return checkpoints, nil
}

// Latest returns the most recent checkpoint, or false if none exist.
func Latest(repoRoot string) (Checkpoint, bool, error) {
	all, err := List(repoRoot)
	if err != nil {
		return Checkpoint{}, false, err
	}
	if len(all) == 0 {
		return Checkpoint{}, false, nil
	}
	return all[len(all)-1], true, nil
}


func Find(repoRoot string, id string) (Checkpoint, error) {
	all, err := List(repoRoot)
	if err != nil {
		return Checkpoint{}, err
	}
	if len(all) == 0 {
		return Checkpoint{}, fmt.Errorf("no checkpoints exist yet")
	}
	if id == "latest" {
		return all[len(all)-1], nil
	}
	for _, cp := range all {
		if strconv.FormatInt(cp.Timestamp, 10) == id {
			return cp, nil
		}
		if strings.HasPrefix(cp.Commit, id) {
			return cp, nil
		}
	}
	return Checkpoint{}, fmt.Errorf("no checkpoint matches %q", id)
}