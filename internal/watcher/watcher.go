package watcher
 
import (
	"os"
	"path/filepath"
	"strings"
	"time"
 
	"git-savepoint/internal/checkpoint"
)
 

type Config struct {
	PollInterval     time.Duration // how often to scan the filesystem
	IdleThreshold    time.Duration // create a checkpoint after this much quiet time following activity
	MaxInterval      time.Duration // always checkpoint at least this often if anything changed
	IgnoreDirs       []string
	IgnoreFilePrefix []string
}

func DefaultConfig() Config {
	return Config{
		PollInterval:  2 * time.Second,
		IdleThreshold: 20 * time.Second,
		MaxInterval:   5 * time.Minute,
		IgnoreDirs:    []string{"node_modules", ".git", "dist", "build"},
	}
}
type Watcher struct {
	repoRoot string
	cfg      Config
 
	fingerprint    string
	lastChangeAt   time.Time
	lastCheckpoint time.Time
	hasPendingWork bool
 

	OnCheckpoint func(msg string, err error)
}
 
func New(repoRoot string, cfg Config) *Watcher {
	return &Watcher{repoRoot: repoRoot, cfg: cfg, lastCheckpoint: time.Now()}
}
 
func (w *Watcher) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
 
	w.fingerprint = w.scan()
 
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			w.tick()
		}
	}
}
 
func (w *Watcher) tick() {
	current := w.scan()
	if current != w.fingerprint {
		w.fingerprint = current
		w.lastChangeAt = time.Now()
		w.hasPendingWork = true
	}
 
	if !w.hasPendingWork {
		return
	}
 
	idleLongEnough := time.Since(w.lastChangeAt) >= w.cfg.IdleThreshold
	overdue := time.Since(w.lastCheckpoint) >= w.cfg.MaxInterval
 
	if idleLongEnough || overdue {
		w.takeCheckpoint("")
	}
}
 

func (w *Watcher) CheckpointNow(reason string) {
	w.takeCheckpoint(reason)
}
 
func (w *Watcher) takeCheckpoint(message string) {
	cp, err := checkpoint.Create(w.repoRoot, message)
	if err == nil {
		w.hasPendingWork = false
		w.lastCheckpoint = time.Now()
	}
	if w.OnCheckpoint != nil {
		w.OnCheckpoint(cp.Message, err)
	}
}
 

func (w *Watcher) scan() string {
	var sb strings.Builder
	filepath.Walk(w.repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(w.repoRoot, path)
		if info.IsDir() {
			for _, ignored := range w.cfg.IgnoreDirs {
				if info.Name() == ignored {
					return filepath.SkipDir
				}
			}
			return nil
		}
		for _, prefix := range w.cfg.IgnoreFilePrefix {
			if strings.HasPrefix(info.Name(), prefix) {
				return nil
			}
		}
		if info.Name() == ".env" || strings.HasPrefix(info.Name(), ".env.") {
			return nil
		}
		sb.WriteString(rel)
		sb.WriteByte(':')
		sb.WriteString(info.ModTime().String())
		sb.WriteByte(':')
		sb.WriteString(time.Duration(info.Size()).String())
		sb.WriteByte('\n')
		return nil
	})
	return sb.String()
}