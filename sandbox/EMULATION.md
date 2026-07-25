# EMULATION — сухая трассировка алгоритма

> Проверка модели из [`../docs/concept.md`](../docs/concept.md) **до написания кода**: провести
> каждый сценарий по трубе руками и убедиться, что алгоритм полон, детерминирован и непротиворечив.
> Базовая пара — [`provider/asyncapi.v1.yaml`](provider/asyncapi.v1.yaml) +
> [`consumer/consumed-contract.yaml`](consumer/consumed-contract.yaml) +
> [`consumer/contract-tests.yaml`](consumer/contract-tests.yaml). Индекс мутаций —
> [`scenarios/scenarios.yaml`](scenarios/scenarios.yaml).
>
> Трассировка **не была формальностью**: она нашла три дефекта модели, каждый из которых
> перепроверен на реальном парсере и привёл к правке концепта. См. §4.

## 0. Что подставляется в трубу

Шаги трубы (`concept.md` §4), сокращённо: `Config → Contract → Spec → Derive → Compare → Report → Exit`.

Проекция базовой спеки поставщика после `DeriveProviderChannels` — то, с чем работает ядро.
Получена **прогоном настоящего парсера** (`lerenn/asyncapi-codegen@v0.63.0` + предрезолв `#/servers/`),
а не выписана от руки:

| address | protocol | направление `receive` (поставщик слушает) | направление `send` (поставщик вещает) |
|---|---|---|---|
| `WALLET.BALANCE.REQUEST` | kafka | `getBalanceRequest` | — |
| `WALLET.BALANCE.RESPONSE` | kafka | — | `getBalanceResponse` |
| `WALLET.AUDIT` | kafka | `auditEvent` | — |
| `WALLET.LEDGER.EVENTS` | kafka | — | `ledgerPosted` |
| `WALLET.FX.RATES` | kafka | — | `fxRateChanged` |

Три вещи, без которых этой таблицы не было бы:

1. **Предрезолв `#/servers/`** — без него `Process()` падает и колонки `protocol` просто нет.
2. **Дефолт сообщений** — ни одна из пяти операций спеки не объявляет `messages`; без разворачивания
   все ячейки направлений пустые.
3. **Разворачивание `reply`** — `receiveBalanceRequest` несёт `reply` на `WALLET.BALANCE.RESPONSE`;
   здесь это дублирует собственную `sendBalanceResponse`, но в спеке без такой операции колонка
   `send` у канала ответа заполняется только отсюда.

Контракт потребителя, сжато: `REQUEST.send{getBalanceRequest}`, `RESPONSE.receive{getBalanceResponse}`,
`AUDIT.send{auditEvent}`, `LEDGER.EVENTS.receive{ledgerPosted}`.

**Инверсия направлений** (`concept.md` §2) читается по таблице: потребитель `send` на
`WALLET.BALANCE.REQUEST` смотрит в колонку `receive` поставщика — и наоборот.

---

## 1. S0 — эталонная трасса (совместимо, exit 0)

```
ConfigStore.Load("contract-tests.yaml")        -> RawConfig                                    ok
NewConfig(raw)                                  -> Config{4 channels, provider.spec_path,
                                                          timeout=30, save_json_report=true}    ok
                                                   spec_path XOR spec_url: ровно один ✓
                                                   каждый адрес из channels есть в контракте ✓
ContractStore.Load("./consumed-contract.yaml")  -> RawContract                                  ok
NewConsumedContract(raw)                        -> Contract{consumer=mq-rest-sync-adapter,
                                                     provenance=wallet-balance@1.4.0,
                                                     4 канала}                                  ok
                                                   contract.consumer == config.consumer.name ✓
BuildSpecLoader({timeout:30})                   -> loader                                       ok
loader.Load({spec_path})                        -> читает файл
                                                 -> InlineServerRefs: 5 подстановок #/servers/broker
                                                 -> parser.FromYAML + Process()                 ok
DeriveProviderChannels(spec)                    -> таблица §0 (5 каналов)                        ok
NewComparison(cfg, contract, pchans)            -> Comparison                                   ok
CompareContracts(comparison)                    -> Outcome{violations: [], uncovered: [...]}
```

Ядро, канал за каналом:

| Канал | R1 | R2 | R3 | R4 | R5 (контравар.) | R6 (ковар.) | R7 | R8 | R9 |
|---|---|---|---|---|---|---|---|---|---|
| `WALLET.BALANCE.REQUEST` send | ✓ | kafka=kafka ✓ | receive есть ✓ | `getBalanceRequest` ✓ | `{clientId,requestId}` ⊆ шлёт ✓; **`locale` не required — пропущен** | n/a | типы string=string ✓ | json=json ✓ | location равны ✓ |
| `WALLET.BALANCE.RESPONSE` receive | ✓ | ✓ | send есть ✓ | `getBalanceResponse` ✓ | n/a | читает `{status,data.{balance,currency}}` ⊆ отдаёт ✓; **`data.updatedAt`, `error` не читаются — не нарушение** | `balance` number/double ✓, рекурсия в `data` ✓ | ✓ | ✓ |
| `WALLET.AUDIT` send | ✓ | ✓ | ✓ | `auditEvent` ✓ | `{eventId,actor}` ⊆ шлёт ✓; `occurredAt` не required ✓ | n/a | ✓ | ✓ | **потребитель не объявил → не проверяется** |
| `WALLET.LEDGER.EVENTS` receive | ✓ | ✓ | ✓ | `ledgerPosted` ✓ | n/a | читает `{entryId,amount,tags}` ⊆ отдаёт ✓; **`postedBy` не читается** | `tags` array→items string, рекурсия ✓ | ✓ | **не объявлен → не проверяется** |

```
FoldReport(outcome, now)
                     -> { schema_version: "1.1",
                          validator: "pinout-asyncapi",
                          interaction: "async",
                          consumer: { name: "mq-rest-sync-adapter" },   # version не эмитим — источника нет
                          generated_at: "<инжектирован сверху, RFC3339>",
                          compatible: true,
                          provenance: {wallet-balance, 1.4.0, sha256:…},
                          errors: [],
                          uncovered_channels: ["WALLET.FX.RATES"] }
ReportWriter.Write   -> compatibility_report.json
cli.ResolveExitCode  -> 0
```

**Что доказывает S0.** Поставщик здесь строго богаче потребителя: необязательное `locale` в запросе,
непрочитанные `data.updatedAt`, `error`, `postedBy` в ответах. Всё это законно и вердикт — `compatible`.
На этой самой паре v1 вернула бы `incompatible`: её `areRequiredFieldsSetsIdentical` требует точного
совпадения множеств. S0 — регрессионный якорь против возврата к идентичности.

`uncovered_channels: ["WALLET.FX.RATES"]` при этом заполнен, а `compatible` остался `true` — проверка,
что информационное поле не течёт в вердикт.

---

## 2. Трассы отклонений

Каждая строка — та же труба; показан **шаг, на котором она сворачивает**, и итог.

### Уровень канала (R1–R4)

| # | Мутация | Шаг решения | `errors[]` | exit |
|---|---|---|---|---|
| **S1** | `balanceRequest.address → WALLET.BALANCE.REQ` | `Compare/ResolveChannel`: адреса `WALLET.BALANCE.REQUEST` нет в проекции | 1 × `CHANNEL_NOT_IN_PROVIDER`, `location: WALLET.BALANCE.REQUEST` | 1 |
| **S2** | `servers.broker.protocol → amqp` | `ResolveChannel`: канал найден, `kafka ≠ amqp` | **4 ×** `PROTOCOL_MISMATCH`, `context:{consumer:kafka, provider:amqp}` | 1 |
| **S3** | удалена `operations.receiveBalanceRequest` | `ResolveDirection`: канал есть, колонка `receive` пуста | 1 × `DIRECTION_NOT_IN_PROVIDER`, `location: WALLET.BALANCE.REQUEST send` | 1 |
| **S3b** | ключ `auditLog.messages.auditEvent → auditEventV2` | `ResolveDirection`: направление есть, ключа `auditEvent` среди сообщений нет | 1 × `MESSAGE_NOT_IN_PROVIDER` | 1 |

S2 — единственная одиночная мутация, дающая **четыре** записи: один сервер обслуживает все каналы.
Она и есть доказательство свёртки: труба не короткозамыкает на первом нарушении, а собирает все.
Если реализация вернёт одну запись — правило замкнуло, это баг.

S3 заодно проверяет разворачивание `reply` с обратной стороны: удалённая операция несла `reply` на
`WALLET.BALANCE.RESPONSE`, но у того канала есть собственная `sendBalanceResponse`, поэтому второе
направление уцелело и лишнего нарушения не появилось. Ровно **одна** запись, а не две.

S3b охраняет [D8](../docs/concept.md#d8-ключ-пары-сообщений--ключ-map-channelmessages-а-не-messagename):
реализация, взявшая ключом `Message.Name`, свалится в `MESSAGE_NOT_IN_PROVIDER` на каждом сообщении —
включая S0, который обязан быть зелёным. Пара сценариев S0+S3b запирает решение с двух сторон.

### Уровень сообщения (R5–R9)

| # | Мутация | Шаг решения | `errors[]` | exit |
|---|---|---|---|---|
| **S4** | `BalanceRequestPayload.required += idempotencyKey` | `CompareMessage`, контравар.: `idempotencyKey ∈ required(provider)`, `∉ fields(sends)` | 1 × `MISSING_REQUIRED_SENT_FIELD`, `location: … send getBalanceRequest payload.idempotencyKey` | 1 |
| **S5** | удалено `BalanceData.properties.currency` | `CompareMessage`, ковар.: `currency ∈ reads`, `∉ properties(provider)` | 1 × `READS_FIELD_NOT_PROVIDED`, `location: … receive getBalanceResponse payload.data.currency` | 1 |
| **S6** | `BalanceData.balance.type → string` | `typesMatch`, рекурсия на 2 уровня: `number/double ≠ string` | 1 × `TYPE_MISMATCH`, `details: expected number/double, got string` | 1 |
| **S7** | `GetBalanceRequest.contentType → application/avro` | `CompareMessage`: потребитель пинил `application/json` | 1 × `CONTENT_TYPE_MISMATCH` | 1 |
| **S8** | `GetBalanceResponse.correlationId.location → $message.payload#/corrId` | `CompareMessage`: потребитель пинил `$message.header#/correlationId` | 1 × `CORRELATION_ID_MISMATCH` | 1 |

S5 — сердцевина ковариантности. Схема поставщика открыта (`additionalProperties` по умолчанию), так
что наивная instance-валидация ответа прошла бы: удалённое поле просто отсутствует, а лишних полей
схема не запрещает. Ловится только сравнением множеств.

S8 попутно проверяет обратную сторону D9: в том же прогоне сообщение `auditEvent` не объявляет
`correlationId` ни у потребителя, ни у поставщика — и **не** даёт нарушения. Ожидается ровно одна
запись; вторая означала бы возврат к правилу v1 «совпадать всегда».

### Паттерны — выводятся, а не задаются

| # | Область | Что проверяется | Итог |
|---|---|---|---|
| **S9** | только `WALLET.AUDIT` (fire-and-forget) | канал без `receives` — ковариантное правило не применяется; `receiveAudit` без `messages` → дефолт | `compatible`, exit 0 |
| **S10** | только `WALLET.LEDGER.EVENTS` (pub-sub) | канал без `sends` — контравариантное правило не применяется; `entryId/amount` в `required` поставщика читаются, `postedBy` не читается; рекурсия по `tags: array of string` | `compatible`, exit 0 |

S9 и S10 — проверка на **ложные срабатывания**: в модели без сущности «паттерн» легко случайно
применить обе variance-проверки к одностороннему каналу. Здесь это дало бы `MISSING_REQUIRED_SENT_FIELD`
на pub-sub-канале (поставщик «требует» `entryId`, а потребитель ничего не шлёт) — очевидный абсурд,
который и ловится этими двумя сценариями. Вместе с S0 (request-reply) три паттерна покрыты.

### Ветки отказа адаптера

| # | Мутация | Шаг | Итог |
|---|---|---|---|
| **S11** | `spec_url` рядом с `spec_path` | `NewConfig` — до всякой сверки | `CONFIG_ERROR`, exit **2**, отчёт **не пишется** |
| **S12** | `consumed_contract_path → ./nope.yaml` | `ContractStore.Load` | `FILE_NOT_FOUND`, exit 3 |
| **S13** | битый YAML / `asyncapi: 2.6.0` | `SpecLoader.Load` → `parser.FromYAML` | `PARSE_ERROR`, exit 3 |
| **S14** | `spec_url` отдаёт 404 | `SpecLoader.Load` → HTTP GET | `HTTP_ERROR`, exit 3 |
| **S15** | медленный `spec_url`, `timeout: 1` | `SpecLoader.Load`, бюджет из `cfg.Settings.Timeout` | `TIMEOUT_ERROR`, exit 3 |

S15 отдельно проверяет **позднюю сборку** `SpecLoader`: таймаут берётся из уже провалидированного
конфига, а не из константы. Если loader соберут раньше `NewConfig`, единственным доступным значением
окажется хардкод и этот сценарий станет недостижим — ровно та ловушка, на которую sync-близнец
наступил и вынес в свой ADR-0005.

---

## 3. Инварианты, проверенные по всему набору

| Инвариант | Как проверен | Статус |
|---|---|---|
| `compatible ⇔ errors == []` | во всех 17 трассах вердикт и `errors[]` согласованы | ✓ |
| Каждый код достижим | 17 сценариев покрывают **все 13** кодов `errors[]` enum + `CONFIG_ERROR` | ✓ |
| Каждый код достижим **ровно одним** путём | каждый код рождается в одном модуле: R1–R4 → `Resolve*`, R5–R9 → `CompareMessage`, io/parse → соответствующий I/O-модуль | ✓ |
| Ни один сценарий не даёт двух кодов на одну причину | S1 не даёт заодно R3 (нет канала — направление не проверяется); S3 не даёт заодно R4; S5 не даёт заодно R7 (нет поля — типы не сравниваются) | ✓ |
| Ковариантность не конфликтует с контравариантностью | ни один канал базовой пары не имеет обоих направлений; на односторонних (S9/S10) применяется ровно одно правило; при двустороннем канале правила смотрят в **разные** колонки проекции и пересечься не могут | ✓ |
| Нарушения сворачиваются, а не замыкают | S2 даёт 4 записи одной мутацией | ✓ |
| Информационное не течёт в вердикт | S0: `uncovered_channels` непуст, `compatible: true`, exit 0 | ✓ |
| `CONFIG_ERROR` не попадает в `errors[]` | S11 сворачивает до рождения отчёта; кода нет в enum схемы | ✓ |
| Детерминизм | обход по отсортированным ключам каналов, направлений, сообщений и полей; в проекции §0 порядок фиксирован | ✓ (требование вынесено в DoD) |
| `generated_at` не ломает детерминизм | единственное недетерминированное поле канона `1.1` инжектится сверху портом часов; в трассах подставляется фиксированное значение | ✓ (`concept.md` D10) |
| `subject` есть у каждой ошибки | у R1–R9 — `<address> [<direction> <message key>]` и он же префикс `location`; у io/parse-кодов — входной артефакт, на котором сломались (S12–S15) | ✓ (`concept.md` D11) |

Порядок правил внутри канала — **иерархический, с прекращением ветки**: не найден канал → направления
не смотрим; нет направления → сообщения не смотрим; нет сообщения → поля не смотрим; нет поля → типы
не сравниваем. Это не короткое замыкание всей трубы (соседние каналы продолжают проверяться), а
отсечение заведомо бессмысленных проверок. Без него одна мутация давала бы каскад производных кодов и
диагностика утонула бы.

## 4. Что трассировка изменила в модели

Три дефекта первой редакции концепта. Каждый перепроверен на настоящем парсере, не по описанию.

**1. Правило R9 было мертво.** Первая редакция гейтила `correlationId` на «канал с обоими
направлениями (request-reply)». Трассировка базовой пары показала: request-reply в AsyncAPI 3.0
выражается операцией с `reply`, и адрес ответа обычно **другой** (`WALLET.BALANCE.REQUEST` против
`WALLET.BALANCE.RESPONSE`). Ни один канал обоих направлений не имеет — условие не срабатывает никогда,
ровно там, ради чего вводилось. Исправлено: R8/R9 срабатывают **от объявления потребителя**, что
согласуется с variance-моделью (что потребитель не пинил — поставщик волен иметь).
→ [D9](../docs/concept.md#d9-r8r9-срабатывают-от-объявления-потребителя-а-не-от-паттерна).

**2. Не было кода на «сообщение не найдено».** Правила покрывали «нет канала» и «нет направления», но
не «направление есть, а такого сообщения нет» — случай проваливался бы в лавину
`READS_FIELD_NOT_PROVIDED` по каждому полю. Добавлено правило **R4 `MESSAGE_NOT_IN_PROVIDER`**;
набор кодов вырос с 8 до 9, сценариев — с 16 до 17.

**3. Ключ пары сообщений был выбран неверно.** Модель предполагала парование по имени сообщения.
Прогон показал, что `Specification.Process()` библиотеки **перезаписывает** `Message.Name`
сгенерированным Go-идентификатором: `name: GetBalanceResponse` → `GetBalanceResponseMessage`
(`pkg/asyncapi/v3/message.go:73`). Ключи map при этом сохраняются. Ключом пары зафиксирован ключ map
`channel.messages`; сценарий S3b поставлен сторожем.
→ [D8](../docs/concept.md#d8-ключ-пары-сообщений--ключ-map-channelmessages-а-не-messagename).

Плюс подтверждено прогоном, а не рассуждением: **предрезолв `#/servers/`** обязателен (без него
`Process()` падает на обеих реальных спеках и на нашей sandbox-спеке) и работает на **стоковой**
библиотеке без форка; **дефолт `messages`** обязателен (не объявлен ни у одной из 5 операций);
**разворачивание `reply`** обязательно (иначе провайдер с request-reply без отдельной send-операции
выглядит как «не отдаёт ничего»).

## 5. Что трассировка НЕ проверяет

Сухая трасса проверяет **связность и полноту модели**, а не реализацию. За её пределами остаётся:
поведение на спеках с `allOf`/`oneOf` (вне MVP), фактическая детерминированность обхода (гарантируется
кодом и требуется в DoD), реальные сетевые тайминги S14/S15 (это компонентные тесты) и точные тексты
`message`. Всё перечисленное закрывается на этапе разработки и не влияет на непротиворечивость модели.
