# Design

Harold goes through a Todoist list one task at a time, *"optimized
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
  (*Filters*). No arguments means Todoist's Today view, `today | overdue`
  (*Filters*).
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

**The card is as tall as its task's text** (Will, 2026-11-11: *"dynamically
size the two boxes to the text contents of the task"*). Each half takes the
lines its text needs once wrapped, and at least one, so a task with no
description still shows a divided card. The card takes at most two-thirds of
the screen, so the ledger keeps a third; past that, the title gets at least
half the room, and text that does not fit ends in `…`. The ledger's lines
shift as the card grows and shrinks from task to task.

**The card's edges carry its labels** (Will, 2026-11-11, over two rounds:
the date *"to the top line"*, and in the bottom line *"query on the right,
keys on the left"*): the task's date and the count (`3 of 127`) in the top
edge; the keys and the query in the bottom edge; the line between the halves
plain. When an edge is too short for both its labels, the date and the query
are kept and the other gives way.

**A blank line separates the ledger from the card** (Will, 2026-11-11: *"a
bit more whitespace between the box and the first entry in the ledger"*).

**The date reads as one of** *"how many days already overdue, today,
tomorrow, the due date"*, *"then add the extra dot rule for recurring
tasks"* (Will, 2026-11-11): `3 days overdue`, `1 day overdue`, `today`,
`tomorrow`, `Mon 16 Nov` (`Mon 4 Jan 2027` in another year), then
`· every monday`, Todoist's own wording of a recurring task's rule. Days are
counted on the calendar from the day the tasks were loaded. A task with a
time shows it after the day (`today 5:30pm`); that was chosen with the
change, not by Will, and none of Will's tasks had one on 2026-11-11. A task
with no date leaves the line plain (`triage.Due.Label`).

Not yet decided, and settled live with Will (*Iterating on the UI with Will*
in `AGENTS.md`): what a ledger line shows; what the screen shows when the
list is empty or exhausted; colour; and any key besides Enter. The
two-thirds limit and the one-line minimum were chosen with the change above,
not by Will.

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
- **The terms are combined with `&`.**
- **No arguments is `today | overdue`**, what Todoist's Today view shows
  (Will, 2026-11-11). The filter `today` alone is strictly due today: on
  2026-11-11 it returned 6 of Will's tasks, `overdue` 121, and
  `today | overdue` 127. `harold today` keeps Todoist's meaning.
- **Tasks come by project, in the sidebar's order**: the Inbox, then the
  project tree top to bottom, each project before its subprojects (Will,
  2026-11-11: "the project order is a match"). **Within a project, by
  priority, then by due date** (Will, 2026-11-11): p1 first (the API's `4`),
  then the earliest date, no date last, and otherwise Todoist's order. On
  one day, a task with a time comes before an all-day one, timed ones by
  time; that tie was chosen with the change, not by Will, and on 2026-11-11
  every one of Will's 126 tasks was all-day. A floating time is read in the
  local zone and a fixed one converted to it (`todoist.SortForTriage`).
  Todoist's own order felt *"quite a
  jumbled feeling order"* (Will, 2026-11-11): Will's 127 switched project 65
  times, and 14, the least 15 projects allow, once sorted. Siblings are
  ordered by `child_order`, a tie broken by `order_key`; on Will's 58
  projects the two never disagreed (`todoist.SidebarOrder`).

`login` and `logout`, alone, are commands rather than filters. The query sent
is shown in the card's bottom edge, so a wrong resolution is visible. Escaping
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
  `tdd_…` client ID it is issued in the keyring, with the name it registered
  under. Each machine has its own. `client_name` is "Harold" and
  `client_uri` the repository: they are what the consent screen shows, and
  Todoist fixes them at registration, so a login whose stored client was
  registered under another name registers again. Until that login, the old
  client keeps refreshing the token it was issued.
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
deletes the tokens. It does not revoke the grant at Todoist, and says so:
both of Todoist's revoke endpoints (*Revoke Access Token Api*, and the
RFC 7009 one) ask for the client's secret, which a public client has none of.
Whether they accept a public client's ID alone is untested, since testing it
revokes a live sign-in; until then the grant is removed in Todoist's
settings.
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
