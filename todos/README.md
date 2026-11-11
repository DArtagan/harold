# Outstanding work

One self-contained spec per step, each meant to be opened cold in its own
session. A file here means the work has not been done. When it is, its permanent
facts go into `docs/`, the spec is deleted, and
[`docs/project_log.md`](../docs/project_log.md) gets a line. See *Project
workflow* in [`AGENTS.md`](../AGENTS.md).

The design every spec builds on is [`docs/design.md`](../docs/design.md). Specs
name the sections they depend on rather than restating them.

## In order

Harold writes nothing to Todoist until a spec says it may. Specs are referred
to by name, not number, so finishing one renumbers nothing.

| # | spec | what | needs |
|---|---|---|---|
| 1 | [flake-install](flake-install.md) | Harold on the PATH from its flake, pinned to a release, in Will's NixOS configuration | a first release |

## Waiting on others

- **Go 1.26.11.** The shell and the flake build with nixpkgs' Go 1.26.10,
  which `govulncheck` reported on 2026-11-11 to carry GO-2026-6859, an
  HTTP/1 server crash fixed in 1.26.11. Harold's only server is the login's
  loopback callback, alive for one sign-in. No nixpkgs had 1.26.11 that day.
  When devenv-nixpkgs has it, `devenv update` and `nix flake update`, and
  `govulncheck` should then find nothing.

## After these

Not yet specified:

- **Triage actions**: what a key does to the task on the card (complete,
  reschedule, move, label), the first writes to Todoist, behind the single
  module `AGENTS.md` requires, and the `data:read_write` scope.
- **The web app on vulcanus**: a server-rendered view over the same `triage`
  model (*Architecture* in the design), its sign-in, and its deployment.

## Writing a spec

- **State what was verified and when.** Anything not checked says so.
- **Record why, not just what**, and record decisions Will made verbatim, so they
  are not relitigated.
- **Point at the design rather than restating it.** Two copies drift.
- **End with done-when conditions measured on the real account or the real
  terminal**, never inferred from Harold's own report.
- **Include the opening prompt.**
- **Correct in place.** When a finding is overturned, rewrite the passage and
  note the wrong turn if it could be repeated.
