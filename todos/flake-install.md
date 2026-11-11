# flake-install: Harold on the PATH, from the flake

## Opening prompt

> Install Harold from its flake into Will's NixOS configuration, pinned to a
> release, so `harold` works from any directory. Read `todos/flake-install.md`
> and the Go paragraph of *Development environment* in `AGENTS.md`. Every
> change to the NixOS configuration is shown to Will before it is made.

## Why

Split out of card-and-ledger (Will, 2026-11-11), which built and checked the
package but held off installing it: *"Hold off on #1 for now."* Until this
lands, `harold` exists only inside the checkout's devenv shell, where it is
the checkout's own (*The dev loop* in the design).

## Scope

- **Where the configuration lives** is not recorded here: ask Will, and note
  it in `CLAUDE.local.md`, which is not committed.
- **The input**: `github:DArtagan/harold` at a release tag, `vX.Y.Z`, not a
  branch, so an install changes only when a release is chosen. There is no
  release yet; the first comes when card-and-ledger's pull request merges and
  release-please's follows it (`docs/releasing.md`).
- **`inputs.nixpkgs.follows`**, or not. Following Will's nixpkgs saves a
  second copy, but the package is built with `buildGo126Module`, matching the
  devenv shell's Go (`AGENTS.md`), and that must exist in the nixpkgs it
  follows. Check, and choose with Will.
- **home-manager or the system**: whichever the configuration already uses
  for user tools.
- Its `vendorHash` is checked by CI's flake job, so a release's flake builds.

## Done when

On Will's machine, outside the checkout:

- `harold --version` prints the release's version, not `dev` or `0.0.0`.
- `harold` shows the Today view, signed in with the keyring entries the
  checkout's `harold login` made, with no new login.
- Inside the checkout's devenv shell, `harold --version` still prints `dev`:
  the shell's own `harold` comes first there.
