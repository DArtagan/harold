# Harold

*Harold: I spent so much time with my todo list, my friends gave it a name.*

Harold for Todoist is a terminal UI for going through a Todoist list one task at
a time. The task under consideration fills a card at the foot of the screen;
pressing Enter sends it up into the ledger, which fills from the bottom like a
teletype, and brings the next task in. It is built for triage and for one
person's workflows, a terminal first and later a web app.

Harold is not created by, affiliated with, or supported by Todoist.

```bash
harold                         # today's tasks
harold inbox                   # a bare word: a project, a label, or a date
harold work today              # several words, all of them
harold '#Work | @waiting'      # Todoist's own filter syntax, passed through
```

- [docs/design.md](docs/design.md): what it does and why
- [docs/releasing.md](docs/releasing.md): how versions are cut
- [todos/](todos/README.md): the work remaining, one spec per step

## Development

```bash
devenv shell                   # Go and the tools the hooks run
devenv test                    # every hook over every file, as CI runs it
go run ./cmd/harold today      # the UI, from the checkout
```

## Licence

MIT

---

Produced in association with Doris.
