# AGENTS.md

Guidance for coding agents working in this repository.

Harold is a terminal UI for triaging a Todoist list one task at a time.
[`docs/design.md`](docs/design.md) is the design: the screen, the filter
arguments, the architecture, how it signs in, and how it is tested. Read the
sections a spec names before changing behaviour.

## Development environment

[devenv](https://devenv.sh/) provides Go, golangci-lint, `gh`, `jq`, `skopeo`
and `tmux`. `direnv` loads it on `cd`; otherwise `devenv shell`.

```bash
go get <module>@latest         # a dependency
go test ./...                  # the tests
go test ./... -update          # rewrite the golden files, then read the diff
treefmt                        # format the tree
devenv test                    # every hook over every tracked file, as CI runs it
harold-dev --demo              # the UI, restarted on the same card on every save
harold login                   # sign in, once per machine
harold today                   # the UI, once
nix build                      # the package installs are built from
```

In the shell, `harold` is this checkout's, built from the working tree on
each run; outside it, `harold` is whatever is installed. Both read the same
keyring entries, so one sign-in serves both.

A change to `go.sum` changes the flake's `vendorHash`: set it to
`lib.fakeHash`, `nix build`, and copy the hash it reports. CI's flake job
fails until that is done.

The hooks are the ones `git commit` runs: treefmt, golangci-lint, `go test`,
and `convco` on the commit message. Files git does not track are invisible to
them; `git add -N` a new file first.

Go is nixpkgs' default version, deliberately not pinned: devenv rebuilds its Go
tools from source for any other, which no binary cache holds (see the comment
in [`devenv.nix`](devenv.nix)).

### Working here as an agent

- **`git diff --no-ext-diff`.** The global git config sends `git diff` through
  difftastic, whose side-by-side output is several times the size.
- **Verify with `devenv test`, not by printing files back.**
- **Filter noisy commands** down to the lines that carry the result.
- **Look at the screen, don't infer it.** A change to what Harold draws is
  checked against its golden file, or in a real terminal through `tmux`
  (*Testing* in the design), never from reading the view code. To read the
  screen while it runs:
  `tmux new-session -d -s h -x 70 -y 20 "bash -c 'devenv shell -- harold-dev --demo'"`,
  then `tmux send-keys -t h Enter` and `tmux capture-pane -p -t h`. tmux runs
  the command in Will's login shell, fish, which rejects bash syntax such as
  `$?`, and a tmux server already running lacks the shell's `PATH`: hence
  `bash -c` and `devenv shell --`. Use `--demo`, so no real task enters the
  session's context.
- **A long build is investigated before it is waited on.** Nix offloads builds
  to remote builders whose logs devenv does not stream, so a build in progress
  and a hung one look alike.

## Rules that protect the todo list

- **Harold writes nothing to Todoist until a spec says it may.** Until then it
  asks for the `data:read` scope only, so Todoist itself refuses a write. When
  writes arrive, every one goes through a single module, and a test holds that
  line.
- **No real task enters the repository.** Fixtures, golden files and
  screenshots use synthetic tasks. A recorded API reply is rewritten to
  synthetic content before it is committed. The repository is public.
- **Tokens never reach a log, a golden file, an error message or the
  repository.** They live in the OS keyring, or in the environment where there
  is no keyring.

## Todoist's brand rules

From *Brand usage* in Todoist's [API documentation](https://developer.todoist.com/api/v1/#section/Developing-with-Todoist/Integrations):
"Todoist" is never Harold's primary name ("Harold for Todoist" is the form);
every description of the app, including the OAuth consent screen's, says it is
"not created by, affiliated with, or supported by Todoist"; and no Todoist
logo, mark or brand colour appears in the UI or the repository.

## Project workflow

Work is specified in [`todos/`](todos/README.md), one self-contained spec per
step, ordered in that README. A spec's filename without `.md` is its **slug**,
which names the branch it is worked on. Work happens on that branch, in a
worktree made with `wt switch --create <slug>`, and lands on `main` through a
pull request. [`.config/wt.toml`](.config/wt.toml) sets the worktree up; a plain
`git worktree add` skips that and shares git hooks with every other checkout.

When a spec is done, whatever it leaves permanently true goes into `docs/`, and
the spec is deleted in the same pull request, with a line added to
[`docs/project_log.md`](docs/project_log.md). A retired slug is never reused.

Commits follow [Conventional Commits](https://www.conventionalcommits.org/);
release-please builds versions and the changelog from them. Releasing is in
[`docs/releasing.md`](docs/releasing.md).

### Iterating on the UI with Will

The screen is designed live, in conversation. Each round is one change, shown
rather than described: a golden file or screenshot of the new screen beside
the old, then Will's verdict. A preference Will states is recorded in the
design verbatim, so the next session does not relitigate it.

### Debugging captures

Anything pulled from a running system, such as API replies or logs, goes under
`captures/<topic>/` at the root of the checkout. `.gitignore` excludes it,
because a capture may hold real tasks. Whatever a conclusion rests on beyond
the session belongs in a spec or in `docs/`, with real content replaced.

## How it is phrased

Comments and docs describe what is, in present tense, and carry the reason where
a reader could otherwise reintroduce a bug. Dates and history belong in specs and
commit messages. A fact lives in one place.
