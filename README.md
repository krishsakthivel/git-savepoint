# git-savepoint

Auto-save your code. It watches a Git repo in the background and quietly
saves snapshots as you work, no commits to look at, nothing to remember to
do. If you ever break something, jump back to any earlier point in seconds.

It doesn't touch your normal commits or `git log`. It just sits underneath,
saving your progress.

## Is this safe to run on my repo?

Yes. git-savepoint stores snapshots using Git's own internal object
database (the same content-addressed storage Git already uses for your
commits). It does not create commits on your current branch, does not
modify `git log` or your branch history, and never pushes anything to
your remote. Snapshots live under refs that are separate from your
branches, so they stay completely out of your way until you ask for
them.

## Install (one time)

**macOS / Linux:**
```
curl -fsSL https://raw.githubusercontent.com/krishsakthivel/git-savepoint/main/install.sh | bash
```

**Windows (PowerShell):**
```
irm https://raw.githubusercontent.com/krishsakthivel/git-savepoint/main/install.ps1 | iex
```

Either one downloads the right build for your machine, puts it on your
PATH, and gets `git-savepoint` working from any terminal. Works with
bash, zsh, and fish.

Close and reopen your terminal afterward, then run `git-savepoint` from
anywhere.

**Don't want to run a script?** Grab the binary for your OS straight from
[the latest release](https://github.com/krishsakthivel/git-savepoint/releases/latest),
then:
1. On Windows, double-click the `.exe` once
2. Open a new terminal and run:
   ```
   git-savepoint install
   ```
3. Close and reopen your terminal

Done.

## Use it

In any Git repo:

```
git-savepoint start --daemon
```

That's it. Just code normally. It saves itself in the background every so
often, or whenever you pause.

**Check on it:**
```
git-savepoint status      # is it running? how many saves so far?
git-savepoint timeline    # see every save point
git-savepoint version     # what version am I running?
```

**Undo / roll back:**
```
git-savepoint restore latest
```
This asks for confirmation first, and always saves your current state
before rolling back, so restoring is never a one-way door.

**Stop it:**
```
git-savepoint stop
```

## No terminal? Just double-click

Double-click `git-savepoint.exe` and it starts watching whatever folder
it's sitting in, right there in a window. Double-click it again to stop.

## FAQ

**Does this mess up my Git history?**
No. Saves are stored separately and never show up in `git log`, never get
pushed, never show up to anyone else.

**Is restoring safe?**
Yes. It always takes a backup of your current state first, so you can
always undo an undo.

**What if I delete the exe?**
Nothing you've already saved is affected. Your save points live inside the
repo itself, not in the exe. You'd just need to reinstall to keep making new
ones.

## Uninstall/Updates

**Uninstallation:**
Uninstalling is simple and easy. Run this command:
```
git-savepoint uninstall
```

Follow the instructions after running that command. git-savepoint should already be uninstalled from PATH when run.

**Updating:**
```
git-savepoint update
```
Checks for a newer release and, if there is one, downloads and installs
it in place. Add `--check` to just see if an update is available without
installing it:
```
git-savepoint update --check
```