# Context Runtime — что это и на что способен сейчас

**Контекст — это не чат и не «ещё один RAG».**  
Context Runtime — ядро управления проектной памятью: оно превращает источники,
артефакты и намерение в **проверяемый, воспроизводимый, привязанный к
источникам** контекст для агентов и инструментов.

Статус: **Lab-ready** + **Stabilization Gate S5 пройден** (2026-07-14).
Контракт для Lab/BFF заморожен; ядро не переоткрывать без measured blocker +
ADR. Интеграция через HTTP API v1 и `pkg/contextkit`
(см. [`docs/lab-gate.md`](../docs/lab-gate.md),
[ADR-0042](../docs/decisions/0042-stabilization-gate.md)).

---

## Самое сильное

1. **ContextPack — центр рантайма**  
   Не «нашли куски и скормили модели», а собранный пакет: что взяли, что
   отклонили, почему, с какими spans, checksums и бюджетом. Модель может
   суммировать — **она не становится источником истины**.

2. **Память проекта, а не бесконечный диалог**  
   Граница изоляции — `project_id`. Индексы, пакеты, прогоны и jobs живут в
   проекте. Пути наружу — только `path_key`, без абсолютных путей хоста.

3. **Гибридный поиск без магии**  
   Exact, sparse (в т.ч. Postgres FTS), hybrid, dense (pgvector) — режимы
   выбираются явно. Нет «вектор всё решит»; есть объяснимый merge кандидатов.

4. **Агент с трассой, а не чёрный ящик**  
   Foreground `agent-run` и background jobs — один путь: pack → model → tool →
   verify → replayable trace. У фоновой работы есть owner и cancel.

5. **Операционная зрелость для продукта поверх ядра**  
   Soft quotas (allow / ask / deny), readiness / degraded без «тихого пустого
   успеха», redaction секретов и PII на Lab-видимых поверхностях, metrics,
   inspect «почему так собрали пакет».

6. **Заменяемые адаптеры**  
   Embedder, Completer, metadata, sparse, dense — через конфиг, не через
   хардкод вендора в домене. Продукт меняет провайдера — ядро остаётся тем же.

---

## На что уже способен сейчас

### Память и индексация

- Ingest корпуса в проект: чанки со spans и checksums.
- Локальное хранение артефактов + metadata (memory или Postgres).
- Dense-индекс (pgvector) и sparse FTS — по включению, не «обязательно всё».
- Repair / rebuild пути для индекса (ops-сценарий).

### Поиск и сбор контекста

- Поиск: `exact` | `sparse` | `hybrid` | `dense` | `hybrid-dense` | `query`.
- **Операторный поиск** (`mode=query`, ADR-0043): `"точная фраза"`,
  `AND/OR/NOT/-`, скобки, `~слово` (морфология), `~"словосочетание"`
  (совпадение по леммам — «железная дорога» находит «железной дороги»),
  `lang:ru`. Термы матчятся по границам токенов, интерпретация видна в
  `query_explain`.
- **Русская морфология**: rule-based `context-lang-ru` (`pkg/lang/ru`, без
  словарей) — анализ с явной неоднозначностью, генерация парадигм для
  расширения запроса, ё→е policy. Словарные движки — снаружи, за тем же
  контрактом.
- Сборка **ContextPack** с бюджетом и объяснимыми scores.
- **Inspect** — JSON «что увидел поиск / pack» для отладки и Lab UX.
- Лингвистические и лексикографические контракты в ядре (простые / harness
  адаптеры); тяжёлые language packs — снаружи, по backlog.

### Агенты и инструменты

- Синхронный agent-run: pack → Completer → tool → verifier → trace.
- Фоновые **jobs**: тот же AgentRun in-process, owner обязателен, cancel через
  context, статус в JSON.
- Смена Completer / Embedder конфигом (`fake` / `localecho` / `http` и т.д.) —
  Lab не вшивает провайдера в свой код.

### Поверхности для продукта

| Поверхность | Зачем |
| --- | --- |
| CLI (`context-dev`) | Локальный proof-loop и ops |
| HTTP (`context-serve`) + API v1 | Lab / BFF / любой клиент |
| Go client (`contextkit`) | Типизированная интеграция без `internal/` |

### Контроль и доверие

- Квоты на ingest / pack / agent-run.
- `/health`, `/v1/ready` — явное degraded, а не «200 и пусто».
- Redaction по умолчанию на `model_text` и preview (корпус не переписывается).
- Metrics и история eval для ops-панелей.

---

## Чем это отличается от «просто RAG»

| Обычный RAG-обёртка | Context Runtime |
| --- | --- |
| Вектор + промпт | Детерминированная память → индекс → hybrid → pack → tool → verify → trace |
| Ответ модели = «истина» | Истина — источники и spans; модель — черновик поверх evidence |
| Скрытый пайплайн | Inspect + replayable trace |
| Вендор вшит в продукт | Адаптеры и API v1; продукт брендирует сверху |
| Нет политики | Quotas, redaction, project isolation, ready/degraded |

---

## Для кого это уже продукт

- **Lab / BFF** — UI и DX поверх замороженного контракта, без импорта `internal/`.
- **Companion / agent products** — своё имя, правила, skills; ядро даёт память,
  retrieval и трассы.
- **Инженеры платформы** — один runtime на несколько downstream-продуктов,
  brand-neutral.

Не позиционируется как: чат-приложение, white-label ChatGPT, или SDK «только
векторный поиск».

---

## Честно: чего ещё нет (и это ок)

Lab Gate ≠ «готово навсегда», но S5 уже закрыт. Дальше — только measured
blocker + ADR: [future-layer.md](future-layer.md),
[adapters-backlog.md](adapters-backlog.md),
[stabilization-roadmap.md](stabilization-roadmap.md).

Коротко по факту кода:

- полноценная multi-tenant auth и fine-grained ACL;
- словарные `context-lang-*` (in-repo: rule-based ru/en, ADR-0043) и
  TEI-lexicon адаптеры;
- DOCX/OCR и богатые binary parsers (thin HTML/PDF есть);
- in-core graph store / full Query AST (consumer patterns: ADR-0040/0041;
  operator subset `mode=query` уже есть — ADR-0043);
- QDrant / Turbopuffer / Tantivy как first-class live adapters;
- fuzzy/trigram как обязательный путь;
- distributed workers / leases / DLQ (single-node schedules + in-process jobs
  уже есть);
- OpenAPI codegen, crawler governance.

Ядро уже **достаточно**, чтобы строить Lab и агентные продукты на
evidence-backed контексте. Stabilization Gate **S5 пройден** — default «не
трогать» без measured blocker + ADR
([ADR-0042](../docs/decisions/0042-stabilization-gate.md),
[stabilization-roadmap.md](stabilization-roadmap.md)).

---

## Куда дальше читать

| Документ | Зачем |
| --- | --- |
| [`docs/README.md`](../docs/README.md) | Навигация: как запустить и интегрировать |
| [`docs/concepts.md`](../docs/concepts.md) | Концепты на английском (норматив) |
| [`docs/lab-gate.md`](../docs/lab-gate.md) | Что заморожено для Lab / после S5 |
| [`../README.md`](../README.md) | Корневое описание Runtime (EN, актуальное) |
| [`README.md`](README.md) | Planning hub: что планировать дальше |
| [roadmap-context-core.md](roadmap-context-core.md) | Архитектурный baseline |

*Документ ориентирован на продуктовое понимание «сейчас». Технические детали и
how-to — в `docs/`.*
