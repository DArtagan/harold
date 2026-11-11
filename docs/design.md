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
- **Harold registers itself with Todoist** (Dynamic Client Registration),
  rather than identifying by a metadata document hosted on GitHub Pages:
  nothing to host or publish. A metadata document can replace it later
  without changing the login flow.
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
`GET /api/v1/tasks/filter` with `limit=200`, the most a page holds, following
`next_cursor` to the end. A query is at most 1024 characters, and this endpoint
refuses Todoist's comma operator (API documentation, read 2026-11-10).

- **An argument with Todoist's operators** (`#`, `@`, `&`, `|`, `!`, `(`,
  `)`, `,`, `:`) is passed through verbatim: `harold '#Work | @waiting'`.
  Combined with other words it is parenthesised, so its own `|` binds inside
  it: `harold work '@waiting | p1'` is `#Work & (@waiting | p1)`.
- **Bare words** are resolved against the account, compared
  case-insensitively, the longest run of words first, so a name with spaces
  needs no quotes: `harold home repairs` is `#Home Repairs`. A project becomes
  `#Name`, a label `@name`, and `inbox` the Inbox project whatever it is
  called. Operators inside a name are backslash-escaped.
- **A run that names both a project and a label is refused**, and the error
  offers both spellings (Will, 2026-11-10: refuse and explain, rather than
  match either or prefer the project).
- **Adjacent words that name nothing stay together** as one phrase, for
  Todoist to read as a date or keyword: `harold work next week` is
  `#Work & next week` (Will, 2026-11-10).
- **The terms are combined with `&`.** No arguments is `today`.

`login` and `logout`, alone, are commands rather than filters. The query sent
is shown in the card's top edge, so a wrong resolution is visible. Escaping
and names with spaces have not yet been checked against Todoist's parser.

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
| `internal/todoist` | The API client: tasks by filter, projects, labels. The one module that calls Todoist's API, with GETs only. | no |
| `internal/auth` | OAuth registration, sign-in, refresh, and the keyring. The one module that POSTs. | no |
| `internal/demo` | Synthetic tasks, for `--demo` and the golden files. | no |
| `internal/tui` | The terminal: wraps `triage` in a Bubble Tea model; `Render` draws the screen as a pure function of its state. | yes |
| `cmd/harold` | Arguments, sign-in, and starting the TUI. | yes |

The web app is planned as a server-rendered view over the same `triage`
model, where a key press or a click is a message to `Update`. Nothing is built
for it yet.

## Signing in

OAuth 2.0 for a native app (RFC 8252): the authorization code flow with PKCE,
the system browser, a loopback redirect, and no client secret.

- **Registration.** On its first login, Harold registers a public client by
  `POST https://api.todoist.com/oauth/register` (RFC 7591; no authentication,
  rate-limited) with `token_endpoint_auth_method: "none"`, and keeps the
  `tdd_…` client ID it is issued in the keyring. Each machine has its own.
  `client_name` is "Harold for Todoist" and `client_uri` the repository: they
  are what the consent screen shows.
- **The redirect** is `http://127.0.0.1:<port>/callback`, on a port chosen
  free at login. Registered without a port, it matches any.
- **Scopes**: Harold asks for `data:read` while it is read-only, so a write
  would be refused by Todoist and not only by Harold. Triage actions will need
  `data:read_write`, and a new consent.
- **Access tokens last an hour** (`expires_in: 3600`) and come with a refresh
  token; refreshing is `grant_type=refresh_token` against
  `https://api.todoist.com/oauth/access_token`.

Measured 2026-11-10 with one probe client, "Harold for Todoist (probe)":
registration accepted plain-`http` loopback redirects for `127.0.0.1` and
`localhost`, the `refresh_token` grant, no secret and `data:read` (201). The
authorize endpoint checks `redirect_uri` before the login page: it passed
`http://127.0.0.1:53682/callback` through to login, and refused a foreign
host with `400 invalid_request`, *Invalid redirect URI*. The documentation
says redirect URIs must be HTTPS, *"localhost allowed for testing"*. The
first real login, by Will on 2026-11-11, was issued a refresh token: `harold
login` reported *"Harold keeps the sign-in fresh by itself"*, which it prints
only when the exchange returned one.

`harold login` runs the flow and stores the client ID and tokens in the OS
keyring (Secret Service on Linux, the Keychain on macOS); `harold logout`
deletes the tokens.
Any other command refreshes an expired token and stores the new pair.
`HAROLD_TODOIST_TOKEN`, when set, is used instead of the keyring and is never
refreshed. It is for where there is no keyring: the cluster, CI, a container.
A personal API token from Todoist's settings works there too.

## Testing

- **Pure logic**: `triage` and `filter`, by table-driven unit tests.
- **The screen**: the model is driven through the messages the program
  would send (a window size, the loaded tasks, key presses) at a fixed size
  over the demo tasks, and its view, stripped of styling, is compared with a
  golden file under `internal/tui/testdata/`. No terminal is involved: the
  view is a function of the model. A golden file is the screen as a person
  reads it, so a change to what Harold draws is a diff in review.
  `go test ./... -update` rewrites them; the diff is read before committing.
  The same test checks every screen fills the window exactly and no line is
  wider than it.
- **The real terminal**: Harold run in `tmux`, its screen read back with
  `tmux capture-pane`, for what a golden file cannot show: the alternate
  screen, key handling, restarts in the dev loop.
- **Sign-in**: against a fake of Todoist's three OAuth endpoints, which
  refuses a client secret and checks the PKCE verifier against its challenge.
- **The client**: against an `httptest` server replaying recorded replies,
  synthetic in content. `TestNothingButAuthWrites` fails if any non-test code
  outside `internal/auth` names an HTTP method that writes.
- **Against the real account**, by hand, for what only it can show: sign-in,
  refresh, and that a filter's tasks match Todoist's own view of that filter.

## The dev loop

`harold-dev [flags] [filter…]`, from the devenv shell, runs Harold and
rebuilds and restarts it on every save under `cmd/` or `internal/`, on the
same card: on a restart Harold writes its position (the query and the
ledger's task IDs) to a file, and the next run replays it over a fresh fetch.
`q` ends the loop and forgets the position. Measured 2026-11-10 in a 70×20
tmux pane with `--demo`: a save is on screen, on the same card, 1.3 s later.
`harold-dev --demo` needs no account.
