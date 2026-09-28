# Форк `yahoo-finance-api`: события дробления в истории цен — инструкция (3.3, DEC-13)

**Дата:** 2026-09-28 · **Зачем:** `design.md` рядом, DEC-13 и Д-2 (решение владельца 28.09, 16:26 — делаем).
**Репозиторий:** `github.com/vadim-melnikov/yahoo-finance-api`, ветка по умолчанию `master`. Сейчас проект сидит на
коммите `dd4b99b8bbaf` (`go.mod` проекта, строка `replace`, 04.03.2026). Модуль форка называется
`github.com/oscarli916/yahoo-finance-api`, `go 1.22.5`.

## Что не так сейчас

`GetHistory` (`history.go:142-171`) шлёт в `v8/finance/chart` только `range`, `interval`, `period1`, `period2`. Закрытие
в ответе пересчитано на дробления, а сами события дробления не запрашиваются и не разбираются. Проекту нужна цена,
как бумага торговалась в тот день (FR-8). Пересчитать её обратно можно только по событиям дробления.

Попутно: при ошибке разбора JSON `GetHistory` зовёт `log.Fatalf` (`history.go:163`). Кривой ответ Yahoo завершает весь
процесс — сервер портфеля вместе с ним.

## Шаги

0. **Ты, один раз:** положи в корень форка два файла и закоммить их в `master` до начала сессии — Claude Code читает
   `CLAUDE.md` при старте:
   * `fork-yahoo-CLAUDE.md` (рядом с этой инструкцией) → `CLAUDE.md`;
   * `fork-yahoo-settings.json` → `.claude/settings.json` — разрешения на `go`, `git`, `gh` и чтение, чтобы сессия не
     спрашивала на каждой команде.

   Правила проекта портфеля туда не переносятся: слои, база, миграции, скиллы архитектуры и спек-процесса к библиотеке
   без зависимостей не относятся. Взяты только стиль команд и правило «ветка — PR в свой `master`». Своё:
   * путь модуля не менять;
   * публичный API, которым пользуется портфель, только расширять;
   * **английский во всём** — код, README, тесты, коммиты, PR (решение владельца 28.09, 16:38: вдруг PR в upstream);
   * правки годятся для upstream как есть, без упоминаний портфеля; `CLAUDE.md` и `.claude/` в такой PR не идут.
1. **Ты:** открой сессию Claude Code в репозитории форка (клон или новая сессия с этим репозиторием) и дай ей задание из
   раздела «Задание для Claude Code» ниже целиком.
2. **Claude Code:** ветка `feat/history-split-events` от `master`, правка и тесты по заданию, `go vet ./...` и
   `go test ./...`. Сетевые тесты форка ходят в Yahoo, как и существующие; офлайн-тесты — на фикстуре.
3. **Ты:** ревью PR в форке и мерж в `master`. Запиши хеш коммита мержа.
4. **Дальше — в проекте, это задача 3.3 у Claude Code (tasks.md):** перевести `replace` на новый коммит и поменять
   адаптер. Руками:
   ```sh
   go mod edit -replace github.com/oscarli916/yahoo-finance-api=github.com/vadim-melnikov/yahoo-finance-api@<хеш>
   go mod tidy
   go build ./... && go test ./internal/infra/yahoo/...
   ```
   `go mod tidy` сам превратит хеш в псевдоверсию `v0.0.0-<дата>-<хеш>`.

Порядок: форк нужен только задаче про цену Yahoo. Остальные задачи 3.3 от него не зависят и могут идти параллельно.

## Задание для Claude Code (репозиторий форка)

Текст — на английском, как и всё в форке. Вставлять целиком.

> This repository is a fork of `yahoo-finance-api` (module `github.com/oscarli916/yahoo-finance-api`). Price history
> must be able to return stock split events. Follow `CLAUDE.md`. Do not break the public API in use today
> (`Ticker.History`, `HistoryQuery`, `PriceData`, `Ticker.Search`, `Ticker.Info`): existing calls must behave exactly as
> before.
>
> 1. **Request.** Add a field `Events string` to `HistoryQuery`. When it is non-empty, `GetHistory` adds the query
>    parameter `events` with that value (`"split"`, `"div,split"`). When it is empty, there is no such parameter and the
>    request is the same as today. Move the building of the query parameters into a function that can be tested without
>    the network.
> 2. **Response.** Add `Events` with parsed `splits` to `YahooHistoryResult`. Yahoo sends splits as an object keyed by
>    timestamp:
>    ```json
>    "events": { "splits": { "1598880600": { "date": 1598880600, "numerator": 4, "denominator": 1, "splitRatio": "4:1" } } }
>    ```
>    `numerator` and `denominator` are numbers (a 1-for-8 reverse split comes as `1` and `8`). The `events` field may be
>    absent altogether.
> 3. **Public type and method.**
>    ```go
>    type Split struct {
>        Date        time.Time // moment of the event, UTC (from "date")
>        Numerator   float64
>        Denominator float64
>        Ratio       string    // "4:1", as Yahoo sent it
>    }
>    // HistoryWithSplits is History plus the splits of the same period, in ascending date order.
>    // It sets Events = "split" itself; History does not change.
>    func (t *Ticker) HistoryWithSplits(query HistoryQuery) (map[string]PriceData, []Split, error)
>    ```
>    A split with a zero `numerator` or `denominator` is a parse error, not a silently skipped entry. The method does
>    not adjust prices: how to apply splits is the caller's decision.
> 4. **`log.Fatalf` in `GetHistory`** (JSON decode error) becomes a returned error; the response body is closed as
>    today. A separate commit.
> 5. **Tests:**
>    * offline, fixture `testdata/chart_with_splits.json` — a recorded or hand-built `v8/finance/chart` response with two
>      events: a `4:1` split and a `1:10` reverse split. Check the parse: two events, ascending, numbers and `Ratio`
>      right, `Date` in UTC;
>    * offline: without `Events` the request has no `events` parameter; with `Events: "split"` it has one;
>    * offline: a response without `events` gives an empty slice, not an error; a split with a zero denominator is an
>      error;
>    * offline: malformed JSON in `GetHistory` returns an error instead of terminating the process. Use
>      `httptest.Server` and a swapped `BASE_URL`; preset `getClient().crumb` in the test, otherwise the client goes to
>      `fc.yahoo.com` for cookies (`client.go`);
>    * live, in the style of the existing tests: `AAPL` for 2020-08-01…2020-09-30 via `HistoryWithSplits` returns one
>      `4:1` split dated 2020-08-31.
>
>    `go vet ./...` and `go test ./...` are green. Add a short `HistoryWithSplits` example to the README.
>
> Branch `feat/history-split-events`; a PR into `master` of this repository, not upstream. Commit messages and the PR
> description in English.

## Как проект применит события (для справки, это уже 3.3)

Адаптер `internal/infra/yahoo` просит историю от `from` до сегодня с дроблениями и берёт закрытия в `[from, to]`. Цена дня
`d` = закрытие × Π(`numerator` / `denominator`) по дроблениям с датой позже `d`: дробление 4:1 даёт множитель 4,
консолидация 1:10 — 1/10. Правило в DEC-13 `design.md`, тесты — там же, §6.2 п. 7.
