# Design

Harold for Todoist goes through a Todoist list one task at a time, *"optimized
for triage and my personal workflows"* (Will). It is a terminal UI first, and
*"in the future growing to be an interactive web-app running on vulcanus"*,
Will's Kubernetes cluster. Harold is not created by, affiliated with, or
supported by Todoist; the brand rules it keeps are in `AGENTS.md`.

## Decisions

Made by Will on 2026-11-10, recorded so they are not relitigated:

- **Bubble Tea, in Go**, over Textual, Ratatui and Ink. Two reasons decided
  it: *"an opinionated, clean interaction model matters"*, and Bubble Tea is
  the one that enforces one (*Architecture*); and it is a chance to learn a
  language other than Python. The web app gets a view of its own whichever
  framework the terminal uses, so the choice did not hang on the web.
- **The arguments are Todoist's filter syntax, with bare words resolved**
  (*Filters*). No arguments means `today`.
- **Read-only to begin with.** Enter moves a task into the ledger and changes
  nothing in Todoist. What triage does to a task is designed later.
- **Public, MIT-licensed.**
- **Sign-in by OAuth, tokens in the OS keyring** (*Signing in*). Will asked
  for *"the most 'proper' route"*. This replaces the first plan, a personal
  API token stored with secretspec: OAuth tokens are rotated by Harold every
  hour, so the user never handles one.
- **Installed as a Nix flake package**, and released as a container image as
  well, matching reelbarrow and bookdrop.
- **Go is nixpkgs' default version**, not pinned (see `devenv.nix`).

## The screen

From the bottom up:

1. **The card**, a box across the foot of the screen divided in two
   horizontally: the task's title in the top half, its description in the
   bottom half.
2. **The ledger**, everything above the card. When Enter is pressed, the
   card's task scrolls up into the ledger, which fills from the bottom upward
   like a teletype, and the next task takes the card.

Not yet decided, and settled live with Will (*Iterating on the UI with Will*
in `AGENTS.md`): what
a ledger line shows; the card's height and how a long description is cut;
what the screen shows when the list is empty or exhausted; colour; and any key
besides Enter.

## Filters

`harold [word…]` becomes one Todoist filter query, sent to
`GET /api/v1/tasks/filter` (`query`, `lang`, `cursor`, `limit`: Doist's
TypeScript SDK, `GetTasksByFilterArgs`, read 2026-11-10).

- **Anything with Todoist's operators** (`#`, `@`, `&`, `|`, `!`, `(`, `)`,
  `,`, `:`) is passed through verbatim: `harold '#Work | @waiting'`.
- **A bare word** is resolved against the account: `inbox` is the Inbox
  project; a project's name, compared case-insensitively, becomes `#Name`; a
  label's becomes `@name`; anything else is left for Todoist to read as a date
  or keyword (`today`, `overdue`, `friday`).
- **Several words** are combined with `&`: `harold work today` is
  `#Work & today`.

Proposed, not yet settled: a word that names both a project and a label; a
project whose name has spaces (one quoted argument, `harold 'home repairs'`);
and adjacent unresolved words that make one date phrase (`next week`), which
`&` would split. The query sent is shown on screen, so a wrong resolution is
visible.

## Architecture

The Elm architecture, which Bubble Tea enforces: one model holds all state;
`Update(model, msg)` returns the next model and any commands; `View(model)` is
a function of the model alone; and nothing does I/O except a command, whose
result comes back as a message.

The packages, so the web app can reuse everything but the view:

| package | what | knows Bubble Tea |
|---|---|---|
| `internal/triage` | The session: the queue, the card, the ledger, and the transitions between them. Pure. | no |
| `internal/filter` | Arguments to a query, given the account's projects and labels. Pure. | no |
| `internal/todoist` | The API client: tasks by filter, projects, labels. The one module that talks to Todoist. | no |
| `internal/auth` | OAuth sign-in, refresh, and the keyring. | no |
| `internal/tui` | The terminal: wraps `triage` in a Bubble Tea model and draws it with Lip Gloss. | yes |
| `cmd/harold` | Arguments, sign-in, and starting the TUI. | yes |

The web app is planned as a server-rendered view over the same `triage`
model, where a key press or a click is a message to `Update`. Nothing is built
for it yet.

## Signing in

OAuth 2.0 for a native app (RFC 8252): the authorization code flow with PKCE,
the system browser, a loopback redirect, and no client secret. From
Todoist's API documentation, read 2026-11-10:

- **A public client needs no registration.** Under *OAuth Client ID Metadata
  Document*, Harold's `client_id` is the HTTPS URL of a JSON document Harold
  hosts, with `token_endpoint_auth_method: "none"`. The URL needs a path, a
  public address, and no redirect. The document's `client_name` and
  `client_uri` are what the consent screen shows.
- **Redirect URIs must be HTTPS**, *"localhost allowed for testing"*. Whether a
  loopback redirect is accepted for everyday use, and with which ports, is
  unverified.
- **Scopes**: Harold asks for `data:read` while it is read-only, so a write
  would be refused by Todoist and not only by Harold. Triage actions will need
  `data:read_write`, and a new consent.
- **Access tokens last an hour** (`expires_in: 3600`) and come with a refresh
  token; refreshing is `grant_type=refresh_token` against
  `https://api.todoist.com/oauth/access_token`.

`harold login` runs the flow and stores the tokens in the OS keyring
(Secret Service on Linux, the Keychain on macOS); `harold logout` deletes them.
Any other command refreshes an expired token and stores the new pair.
`HAROLD_TODOIST_TOKEN`, when set, is used instead of the keyring and is never
refreshed. It is for where there is no keyring: the cluster, CI, a container.
A personal API token from Todoist's settings works there too.

## Testing

- **Pure logic**: `triage` and `filter`, by table-driven unit tests.
- **The screen**: `teatest` drives the whole program at a fixed terminal size
  over synthetic tasks, and compares the final screen with a golden file
  under `testdata/`. A golden file is the screen as text, so a change to what
  Harold draws shows up as a diff in review. `go test ./... -update` rewrites
  them; the diff is then read before committing.
- **The client**: against an `httptest` server replaying recorded Todoist
  replies, rewritten to synthetic content before they are committed.
- **Against the real account**, by hand, for what only it can show: sign-in,
  refresh, and that a filter's tasks match Todoist's own view of that filter.
