Слои завершены и проверены. Итог по стандарту stewardship:

## Что реализовано

**1. Русская морфология — `context-lang-ru` (ADR-0043)**
- `pkg/lang/ru` — публичный rule-based движок без словарей: таблицы склонений (7 именных парадигм) и спряжений (6 глагольных + возвратные), анализ с **явной неоднозначностью** («дорогой» → кандидаты и *дорога*, и *дорогой*), генерация парадигм для расширения запроса, ё→е policy (`nfc-lower-yofold-v1`), правило правописания после г/к/х/ж/ш/ч/щ, малый exception-fixture (бежать, идти, человек, ребёнок, год).
- Проходит **публичный** `langtestkit.RunContract` — ровно так, как проходил бы внешний репозиторий.
- `internal/linguistic/ru` + `internal/linguistic/registry` (en/ru) — единая точка выбора адаптера для рантайма.

**2. Операторный поиск — `internal/retrieval/querylang`**
- `"фраза"`, `AND`/`OR`/`NOT`/`-`, скобки, `~слово` (морф-расширение), `~"словосочетание"` (совпадение по последовательности лемм: запрос «железная дорога» находит «вдоль железной дороги»), `lang:ru`.
- Термы матчатся **по границам токенов**: `дом` больше не находит «Домашний» — точность, которой не давал substring-режим.
- Полная объяснимость: канонизированное дерево, принятые/отклонённые расширения, счётчики совпадений в `query_explain` + trace-события; sparse/FTS только усиливает скоринг и не расширяет детерминированное множество.

**3. Подключение (additive v1):** CLI `search --mode query --lang ru`, `CONTEXT_LANG`; HTTP `POST /v1/search` (`mode:"query"`, `lang`, `query_explain`); `contextkit` (`SearchRequest.Lang`, `SearchResult.QueryExplain`). Hybrid-режим тоже получил морф-расширение (заодно исправлен hardcoded `"en"` в hybrid-движке — найденная архитектурная неточность).

**4. Golden gate v3:** RU inflection recall, неправильный глагол, AND/NOT, лемма-фраза, точность границ токенов, ё→е, OR/группировка — с новыми `forbid_chunk_ids` precision-guard'ами. Proof перегенерирован в `.proofs/eval/golden.json`.

**5. Процесс:** ADR-0043 написан через reopen-условие, которое сам ADR-0041 и предусматривал («ship a minimal AST with golden tests and trace payloads»); обновлены ADR-0041, changelog v1, lab-gate, search-docs (новый `docs/search-operators.md`), adapters-backlog, future-layer (L04/L05A), plugins-roadmap и RU-снапшот продукта.

## Проверено

- `go vet` чисто, линтер чисто; **весь `go test ./...` зелёный офлайн**.
- **С живым Postgres (Docker):** metadata/FTS/pgvector интеграции + новый интеграционный тест операторного поиска поверх живого FTS — зелёные (73 пакета ok).
- Lab smoke + golden v3 + adversarial — зелёные; живое CLI-демо: `~"железная дорога" -чат` возвращает единственный правильный chunk со сниппетом, highlight-спанами и explain.

## Отложено сознательно

- Словарная морфология (OpenCorpora-масштаб) — внешний `context-lang-ru` за тем же registry-контрактом.
- Proximity (`NEAR/n`), field-синтаксис, wildcards — вне грамматики до measured-запроса (фильтры остаются в `RetrievalFilters`).
- Posting-list/FTS-pushdown для операторных термов на больших корпусах — follow-up ADR при измеренном блокере (сейчас скан in-memory индекса, что соответствует PoC-масштабу).

## Известный риск

Rule-based анализ пере-генерирует формы на чередующихся основах (бег-/беж- вне fixture, беглые гласные). Контроль: confidence-floor, cap кандидатов, `RejectExp`, golden precision-guards. Правильное решение — словарный адаптер, интерфейс для него готов и доказан harness'ом.