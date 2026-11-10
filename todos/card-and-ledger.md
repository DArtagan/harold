# card-and-ledger: real tasks, one at a time

## Opening prompt

> Build Harold's first end-to-end loop: `harold login`, then `harold [filter]`
> fetching real tasks into the card and the ledger, read-only, with a dev loop
> fast enough to design the screen live with Will, and a flake package to
> install it. Read `todos/card-and-ledger.md`, then all of `docs/design.md`,
> and *Rules that protect the todo list*, *Todoist's brand rules* and
> *Iterating on the UI with Will* in `AGENTS.md`.

## Why

Will chose one spec, end to end, over a UI-first spec on synthetic tasks
(2026-11-10), and sign-in first: nothing reads Todoist until `harold login`
works. The screen in the design is a starting point; *"from there we'll
iterate live on the design and functionality"* (Will), so the dev loop is part
of the work, not an extra.

## Scope, in order

1. **Sign-in** (*Signing in*). Before writing it, confirm what the design marks
   unverified: that Todoist accepts a loopback redirect, on which ports, and
   that a metadata-document client is issued a refresh token. Try it with a
   throwaway document before building around it, and record the answer in the
   design.
   - The client metadata document, served from GitHub Pages at a URL under
     `https://dartagan.github.io/harold/` (no custom domain redirects it,
     checked 2026-11-10). `client_name` "Harold for Todoist"; `client_uri` the
     repository. Enabling Pages is a change to the repository's settings:
     confirm it with Will.
   - `harold login` and `harold logout`; refresh on use; the keyring; the
     `HAROLD_TODOIST_TOKEN` override.
   - The scope is `data:read`.
2. **The client and the filters** (*Filters*): tasks by filter with its
   cursor followed to the end, projects and labels for resolving bare words.
   Settle the design's proposed cases with Will and write the answers into
   *Filters*.
3. **The card and the ledger** (*The screen*), in `internal/triage` and
   `internal/tui`, on Bubble Tea v2 (`charm.land/bubbletea/v2`) and Lip Gloss
   v2. The query sent to Todoist is visible on screen. Golden-file tests from
   the first screen on (*Testing*).
4. **The dev loop**, before the first design round:
   - A save rebuilds and restarts Harold in its terminal within a couple of
     seconds (`watchexec` or similar, from devenv), and the terminal is
     restored cleanly when the old process is stopped.
   - A restart returns to the same place: the same filter, the same task on
     the card, the same ledger. A dev-only snapshot of the model, written on
     exit and read on start, is one way; the Elm model makes it a matter of
     serialising one value.
   - `harold --demo` draws synthetic tasks with no account, for golden files,
     screenshots, and design rounds that should not show real tasks.
   - A way to put a screen in front of Will away from a terminal: a golden
     file is text; a `vhs` tape gives an image. Choose with Will.
5. **The flake package**: `flake.nix` exporting `packages.default`, built
   with `buildGoModule`, version from the release tag. Its `vendorHash` has to
   change with every dependency change; a hook or CI job that builds it keeps
   that from being found at install time.

## Bindings that are easy to miss

- Bubble Tea's v2 module path is `charm.land/bubbletea/v2`, not the
  `github.com` one: v2.1.1 was current on 2026-11-10. `teatest` for v2 is
  `github.com/charmbracelet/x/exp/teatest/v2`, unversioned.
- No real task in any golden file, fixture or screenshot that is committed
  (*Rules that protect the todo list*). `--demo` exists partly for this.
- A recorded API reply is rewritten to synthetic content before it is
  committed. Its shape is what the test needs, not its content.

## Done when

Measured on Will's account and terminal, not inferred:

- `harold login`, from an empty keyring, opens the browser, the consent screen
  names "Harold for Todoist" and asks for read access only, and afterwards the
  tokens are in the keyring and nowhere on disk.
- More than an hour later, `harold` still works without a new login.
- A write attempted with Harold's token, by hand with `curl`, is refused by
  Todoist.
- `harold`, `harold inbox`, a project, a label, a date and a pass-through
  query each show the tasks Todoist's own app shows for the same filter.
- Enter moves the card's task into the ledger and brings the next, and Will
  has gone through the real Today list with it.
- An edit to the view, saved while Harold runs, is on screen with the same
  task on the card in under five seconds.
- Harold, installed from the flake in Will's NixOS configuration, runs
  `harold --version` and `harold today` outside the checkout.
