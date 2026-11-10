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
| 1 | [card-and-ledger](card-and-ledger.md) | `harold login`, then `harold [filter]` showing real tasks in the card and the ledger, with a dev loop fast enough to design the screen live; installed from the flake | — |

## After these

Not yet specified, each waiting on what card-and-ledger teaches:

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
