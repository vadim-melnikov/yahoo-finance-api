# yahoo-finance-api — fork for the Portfolio project

## What this repository is

- A fork of `github.com/oscarli916/yahoo-finance-api`, owned by `vadim-melnikov`. It has one consumer: the owner's
  Portfolio project (a Go service), which pulls it through a `replace` directive in its `go.mod`.
- The module path stays `github.com/oscarli916/yahoo-finance-api`. Never change it: the consumer's imports and its
  `replace` depend on it.
- Work goes into this fork's `master` through pull requests. Do not open a PR against the upstream repository unless the
  owner asks for it.
- Keep every change upstream-friendly, so it could later be offered to upstream as is: no references to the Portfolio
  project in code, comments, tests or README. `CLAUDE.md` and `.claude/` are fork-local and stay out of any upstream PR.

## Compatibility contract

The Portfolio project calls the API below. It must keep compiling and behaving as before — add to it, never change or
remove:

- `NewTicker(symbol)`, `(*Ticker).Search(query, limit)`, `(*Ticker).Info()`, `(*Ticker).History(HistoryQuery)`;
- `HistoryQuery{Start, End, Interval}` with dates as `YYYY-MM-DD`;
- the fields of `PriceData`, `SearchResult` (`Symbol`, `Exchange`, …) and `YahooTickerInfo`.

If a task seems to require changing any of these, stop and ask the owner.

## Coding

- Go version as in `go.mod`. Standard library only: the module has no dependencies, keep it that way.
- Library code returns errors. No `log.Fatal`, `os.Exit` or `panic`: the consumer is a long-running server.
- Keep the style of the file you touch; `gofmt` everything.
- English everywhere: code, comments, README, test names, commit messages, PR titles and descriptions — even when the
  task text arrives in another language.

## Tests

- `go vet ./...` and `go test ./...` pass before every commit.
- The existing tests call the live Yahoo API. New behaviour gets OFFLINE tests first: `httptest.Server` plus JSON
  fixtures in `testdata/`, with `BASE_URL` swapped and `getClient().crumb` preset, so the client does not go to
  `fc.yahoo.com` for cookies (`client.go`). A live test is a supplement in the style of the existing ones, never the only
  test of a behaviour.
- The plain `testing` package, no testify: no new dependencies.

## Workflow

- One branch per task (`feat/<short-name>`), one commit per finished step, a PR into `master` of this fork:
  `gh pr create --repo vadim-melnikov/yahoo-finance-api --base master`.
- The task text comes from the Portfolio project's specs; the owner pastes it into the session.
- After the merge the Portfolio project moves its `replace` to the merge commit. That happens in the Portfolio
  repository, not here.

## Shell command style

Write commands flat, so the permission system can parse them without asking:

- no command substitution `$(...)` or backticks — pipe into `xargs` instead;
- no shell variables — inline the literal value;
- no `cd` inside compound commands — use absolute paths;
- no subshells `( ... )`, `|| true` guards or heredocs — cap output with `head`/`tail`;
- no `for`/`while` loops in one command;
- multi-line commit messages via several `-m` flags or `-F <file>`.
