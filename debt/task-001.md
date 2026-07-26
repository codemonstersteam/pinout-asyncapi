# DEBT task-001: узлы пайпа с двумя входами + две непокрытые ветки

> **Тип:** технический долг · **Вес:** `patch` — поведение не меняется, отчёт байт-в-байт прежний
> **Срез:** `slice-01-validate` (`internal/validate/`)
> **Источник:** ревью после первого прогона; правило дизайна — `docs/design/slice-01-validate/module-tree.md` §4,
> `contracts.md` (антецедент → консеквент), `docs/concept.md` D10.

## Правило, которое чиним

**У узла пайпа ровно один вход данных — свой для этого шага.** Окружение (порты, часы, HTTP-клиент,
уже провалидированные скаляры из конфига) связывается **до** пайпа, в композиционном корне, и входит в
шаг готовым коллаборатором с одноаргументным методом. Настоящий join нескольких потоков материализуется
отдельным **именованным** входным типом.

Идиом в коде уже есть и работает — его и надо распространить:

```go
loader := BuildSpecLoader(cfg.provider, cfg.settings.Timeout, deps.HTTPClient)  // связали до пайпа
spec, err := loader.Load()                                                       // в пайпе — без аргументов
writer := BuildReportWriter(cfg.settings)
report, err = writer.Write(report)                                               // один аргумент данных
```

## Что переписать (три места)

| сейчас | станет | почему |
|---|---|---|
| `FoldReport(outcome Outcome, clock Clock) Report` | `BuildReporter(clock Clock) Reporter` + `Reporter.Fold(Outcome) Report` | `clock` — порт, а не домен. Тот же приём, что у `BuildReportWriter` |
| `NewConsumedContract(raw RawContract, expectedConsumer string)` | `BuildContractParser(consumerName string) ContractParser` + `.Parse(RawContract) (ConsumedContract, error)` | `expectedConsumer` — уже провалидированный скаляр из конфига, часть настройки шага, а не поток данных |
| `NewComparison(cfg Config, contract ConsumedContract, pchans ProviderChannels)` | `NewComparison(in ComparisonInput) (Comparison, error)`, где `ComparisonInput{Config, Contract, ProviderChannels}` | это **настоящий join** трёх независимых потоков. Убрать его нельзя — можно назвать: шов получает имя, тесты строят одно значение |

После этого `ProcessValidate` читается как блок сборки коллабораторов + линейная цепочка одноаргументных
шагов, и «documented exceptions» по арности в дизайне не остаётся ни одного.

## Анти-паттерн — не превращать правило в его противоположность

**Запрещено** вводить общий `Context`/`State`, протаскиваемый через весь пайп. «Один вход» ≠ «один общий
объект»: общий контекст даёт каждому узлу доступ ко всему, антецедент перестаёт быть узким, изолированно
проверить шаг нельзя, а таблица «антецедент → консеквент» в `contracts.md` превращается в фикцию.
У каждого шага — **свой** узкий входной тип.

## Две непокрытые ветки (чинить вместе с рефакторингом)

1. **`internal/validate/adapter.go` без юнит-тестов.** Там `cli.Parse(args)` и `ResolveExitCode(report, err)`
   (`adapter.go:32`) — а последний дизайн сам называет единственным местом, где класс ошибки превращается
   в exit-код. Единственное место маппинга не проверено ничем.
2. **Exit-код `1` не проверен нигде.** `component-tests/features/validate.feature` покрывает `0`, `2` и
   шесть раз `3`; `cmd/app/main_test.go` — только `2`. То есть **главный вердикт инструмента**
   («несовместимо» → `1`) сквозь весь бинарь не доказан. Юниты доказывают, что `Outcome` несёт нарушения,
   но переход `Outcome → exit 1` — ничем.

`ResolveExitCode(report, err)` под правило арности **не подводить**: это распаковка `Result` на границе,
в Go нет суммарных типов.

## Границы

- `api-specification/` заморожен — не трогать; формат отчёта не менять.
- D10 обязан сохраниться: системные часы читаются ровно один раз, `Now()` вызывается единожды — теперь
  внутри `Reporter`. Детерминизм «те же входные байты → те же байты отчёта» проверяется.
- `docs/design/slice-01-validate/{module-tree,contracts}.md` обновить под новые сигнатуры: дизайн —
  источник правды, он не имеет права разойтись с кодом.

## Definition of Done

- [ ] `go build ./...`, `go vet ./...`, `go test ./...` — зелёные.
- [ ] В `head.go` ни одного шага с двумя аргументами данных; коллабораторы (`reporter`, `parser`, `loader`,
      `writer`) собраны до цепочки.
- [ ] `ComparisonInput` объявлен в `contracts.md` как join; раздел «documented exceptions» по арности пуст.
- [ ] `internal/validate/adapter_test.go`: `Parse` (happy + Σ ветвей антецедента) и `ResolveExitCode` —
      все четыре строки грида (`0` compatible · `1` incompatible · `2` config · `3` io/parse).
- [ ] `validate.feature`: новый сценарий на **несовместимую** пару — `exit 1`, отчёт с непустым `errors[]`.
- [ ] `component-tests/scripts/run-tests.sh` зелёный целиком.
- [ ] Отчёт на базовой совместимой паре байт-в-байт совпадает с текущим (рефакторинг ничего не сдвинул).
