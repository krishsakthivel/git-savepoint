# v1.2.1
- Easier installation instructions.

# v1.2.0
- Added version command
- Added update command

# v1.1.1/2
- README updates
- Automatic binary releases on commit (includes Linux distro)

# v1.1.0

- Added tests
- Fixed bug: checkpoint timestamps are stored with 1-second resolution `time.Now().Unix()`, and refs are keyed by that timestamp. Two checkpoints taken within the same second get the same ref name, so the second one silently overwrites the first. (if we add manual checkpointing in the future, this is BAD.)
- Added repo state detection
- Added MIT license
- Improved README
- Uploaded changelog

# v1.0.0/1

- Made installation easier.
- Added uninstall.


# Note beyond this. 

Below this are alpha releases. Sadly, I was too lazy to make changelogs for them before, so I made makeshift changelogs based on what I remembered. 

# v0.9.0-alpha

- Fixed windows + linux installation.
- `git-savepoint` is set on PATH automatically

# v0.8.0-alpha

- Added manual instructions for how to set `git-savepoint` on PATH.
- Added a way to set `git-savepoint` on PATH.

# v0.7.0-alpha

- Created `watcher.go`
- Implemented state-based background worker (`Watcher.Run`) that polls for filesystem changes.
- Added adaptive checkpoint triggers based on configurable quiet periods (`IdleThreshold`) and hard backup deadlines (`MaxInterval`).
- Implemented file tree hashing via filepath serialization of mod times and sizes.
- Added native path exclusion rules for common dependency directories and sensitive `.env` files.

# v0.6.0-alpha

- Created `storage.go`
- Created `Checkpoint` struct to encapsulate snapshot metadata (timestamp, commit hash, and user/auto-generated message).
- Implemented `Save()` utilizing `git update-ref` to write checkpoint pointers securely under `refs/git-savepoint/checkpoints/`.
- Implemented `List()` utilizing `git for-each-ref` to retrieve, parse, and sort saved checkpoints in ascending order.
- Added `Latest()` helper for rapid retrieval of the most recent savepoint.
- Added `Find()` matching logic supporting lookup by strict timestamp IDs, "latest" aliases, or commit hash prefixes.

# v0.5.0-alpha

- Created `restore.go`
- Implemented `To()` to find a target checkpoint and restore the repository's tracked files to match it.
- Integrated automated safety backups by calling `checkpoint.Create()` to protect uncommitted changes before performing the restore.
- Added graceful handling for clean working directories (`checkpoint.ErrNoChanges`), allowing restores to proceed without creating redundant safety checkpoints.
- Utilized `git checkout` to securely update the local filesystem to match the specified checkpoint commit.

# v0.4.0-alpha

- Created `gitutil.go`
- Implemented `Run()` to execute Git subprocesses asynchronously within a specified directory, capturing and returning trimmed stdout or formatting stderr on failure.
- Implemented `RunWithEnv()` to support running Git commands with custom environment variables injected alongside the host environment.
- Added `RepoRoot()` to find the absolute path of the workspace's root directory using `git rev-parse --show-toplevel`.
- Added `GitDir()` to resolve the path of the `.git` metadata directory safely, handling both relative and absolute paths.
- Added `HeadCommit()` to easily query the current `HEAD` commit hash, gracefully returning an empty string on error or unborn branches.

# v0.3.0-alpha

- Created `checkpoint.go`
- Implemented `Create()` to snapshot and persist the workspace's current state.
- Utilized a temporary scratch index (`git-savepoint-index.tmp`) via `GIT_INDEX_FILE` to stage files independently, preventing interference with the user's primary Git index.
- Integrated default exclusion rules (`DefaultIgnore`) using Git's pathspec magic `:(exclude)` to automatically skip node modules, build artifacts, `.env` configurations, and logs.
- Added change-detection logic that compares the newly generated tree hash with the previous checkpoint to avoid duplicate writes, returning `ErrNoChanges` when the working tree is unchanged.
- Implemented linear commit parent linking (`commit-tree`), dynamically linking the new checkpoint to the previous checkpoint, or falling back to `HEAD` for the initial snapshot.
