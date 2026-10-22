# Postgres: реестр StakeDAO

Миграция выполняется вручную **один раз**, до запуска приложения. Код не создаёт
таблицы автоматически. При настроенном `DATABASE_URL` и отсутствующей миграции
сервер завершает запуск с понятной ошибкой, без переключения на полное сканирование.
Поддерживается PostgreSQL 17; драйвер Go — `pgx/v5`, пул до 5 соединений.

## Установленный Postgres: вручную через pgAdmin

Docker не требуется. Используйте установленный PostgreSQL и свой существующий
логин. Пароль и адрес подключения задаются только на вашем компьютере.

1. В pgAdmin подключитесь к своему серверу PostgreSQL.
2. Создайте базу `abievm` через **Databases → Create → Database** либо выберите
   уже созданную базу. Владельцем укажите своего пользователя.
3. Именно для этой базы откройте **Query Tool**.
4. Откройте файл `migrations/000001_registry.up.sql` и выполните его **целиком**,
   вместе с `BEGIN` / `COMMIT`. Скрипт уже содержит все таблицы, индексы и ограничения;
   вручную собирать колонки через GUI не нужно.
5. Обновите дерево **Schemas → public → Tables**: должны появиться
   `registry_sync_state`, `registry_checkpoints`, `registry_events`.

Применяйте скрипт один раз. Он не создаёт саму базу или пользователя. Подключение
приложения должно указывать на ту же базу, где выполнен скрипт. Для простого
локального запуска используйте тот же логин, которым создавали таблицы; при
отдельном пользователе приложения ему нужны права чтения/записи этих таблиц.

Вариант без pgAdmin — установленный `psql`, из корня проекта:

```powershell
$env:DATABASE_URL = "postgres://postgres:YOUR_PASSWORD@127.0.0.1:5432/abievm?sslmode=disable"
psql $env:DATABASE_URL -v ON_ERROR_STOP=1 -f migrations/000001_registry.up.sql
```

Замените `postgres`, `YOUR_PASSWORD`, порт и имя базы своими значениями.
Все реквизиты уже содержатся в `DATABASE_URL`; отдельные `POSTGRES_USER` /
`POSTGRES_PASSWORD` для Go-приложения не нужны. Спецсимволы в пароле URL нужно
URL-encode, например `@` → `%40`, `#` → `%23`. Для локального примера показан
`sslmode=disable`; для удалённого Postgres укажите требуемый сервером TLS.

## Подключение Go/Echo

```powershell
$env:DATABASE_URL = "postgres://postgres:YOUR_PASSWORD@127.0.0.1:5432/abievm?sslmode=disable"
$env:STAKEDAO_ETHEREUM_RPC_URL = "ВАШ_ETHEREUM_MAINNET_RPC"
$env:RPC_MAX_RPS = "18"
$env:STAKEDAO_DISCOVERY_WORKERS = "8"
go run -mod=readonly ./cmd
```

Если запускаете из GoLand, задайте их в
**Run → Edit Configurations → ваша конфигурация → Environment variables**:

```text
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@127.0.0.1:5432/abievm?sslmode=disable
STAKEDAO_ETHEREUM_RPC_URL=ВАШ_ETHEREUM_MAINNET_RPC
```

Логин и пароль в первой строке — ваши реальные реквизиты. При успешном старте
в консоли появится `StakeDAO registry storage backend=postgres`. После этого
используйте прежний Swagger `GET /reservesStakedao/ethereum`.

Переменные другой PowerShell-сессии не попадут в уже запущенную IDE.
`.env.example` можно использовать как образец; сам Go `.env` не читает.
Echo не требует отдельного DB-плагина: `cmd/main.go` открывает пул, проверяет схему
и передаёт репозиторий через `NewServerWithRegistry` → `NewHandlerWithRegistry` →
`NewStakeDAOServiceWithRegistry`. При завершении Echo пул закрывается.

Если `DATABASE_URL` не задан, сохраняется прежний режим без БД: полный endpoint
заново сканирует историю. Режим явно печатается при старте.
Если URL задан неверно или Postgres недоступен, тихого fallback нет.
Проверка `cmd/stakedao-check -limit 5` по-прежнему идёт напрямую по gauge без БД.

## Необязательный вариант на будущее: Docker

Для вашей установленной БД этот раздел можно пропустить. `compose.postgres.yml`
оставлен как альтернативный способ запуска отдельного локального сервера:

```powershell
$env:POSTGRES_PASSWORD = "change_me_local"
docker compose -f compose.postgres.yml up -d --wait
Get-Content -Raw migrations/000001_registry.up.sql | docker compose -f compose.postgres.yml exec -T postgres psql -U abievm -d abievm -v ON_ERROR_STOP=1
```

В этом варианте создаются отдельные база/пользователь `abievm`, поэтому URL будет
`postgres://abievm:change_me_local@127.0.0.1:5432/abievm?sslmode=disable`.
Если установленный Postgres уже занимает 5432, для контейнера нужен другой
`POSTGRES_PORT`. Docker Compose может читать `.env`, Go автоматически его не читает.
Пароль существующего Docker volume не меняется от простой замены переменной
POSTGRES_PASSWORD после первого создания базы.

## Файлы и обязанности

- `internal/service/stakedao/registryService.go`: finalized header, проверка
  курсора/hash, чтение недостающих диапазонов, сохранение прогресса, откат и
  добавление временного незавершённого участка до snapshot-блока.
- `internal/service/stakedao/registryStore.go`: интерфейс хранилища и типы.
- `internal/repository/stakedao/registryPostgres.go`: SQL, транзакции,
  чтение последнего блока и событий, блокировка одного потока синхронизации.
- `000001_registry.up.sql`: таблицы и индексы; `down.sql`: удаление схемы.

В БД три таблицы:

| Таблица | Данные |
| --- | --- |
| `registry_events` | История `VaultRegistered`, `GaugeShutdown`, `GaugeUnshutdown`, с адресами и координатами событий |
| `registry_sync_state` | Начальный и последний **полностью обработанный** блок, его hash и версия декодера |
| `registry_checkpoints` | Последние 16 контрольных точек для ограниченного отката |

Это события реестра, не вся история сети. Текущая регистрация вычисляется по
последнему `VaultRegistered` каждого gauge; история изменений сохраняется.
Shutdown-история также хранится, а актуальный статус, vault, asset, receiver и
protocolId проверяются через `Controller.gauge` на блоке расчёта. Неактивные
пулы пропускаются. Цены, TVL и APR в эти таблицы не записываются.
Ключ потока — chain ID + platform + Controller. Цена токенов остаётся в map.

## Как работает продолжение

1. Сервис читает последний курсор и сверяет сохранённый hash с RPC.
2. Параллельно читает диапазоны до 1500 блоков, но сохраняет их строго по порядку.
   Размер очереди ограничен workers; через непрочитанный диапазон не перескакиваем.
3. События, новая контрольная точка и курсор фиксируются одной транзакцией.
   Пустой диапазон тоже продвигает курсор.
4. При ошибке RPC, нулевых CU или таймауте уже зафиксированные диапазоны остаются.
   Следующий полный запрос продолжит с `last_scanned_block + 1`.
5. При одинаковом finalized-блоке старая история не перечитывается. Между finalized
   и snapshot остаётся небольшой свежий участок: он читается заново и не сохраняется
   как окончательная история. Поэтому новые пулы доступны и до финализации.

RPC должен поддерживать Ethereum-тег `finalized`. При его отсутствии сервис
возвращает ошибку, а не выбирает произвольное число подтверждений. Индексатор пока
подключён только к StakeDAO Ethereum; для других платформ нужны свои адаптеры.

Контрольные hash проверяются на концах диапазонов и в блоках найденных событий.
При реорганизации откатываются **события и курсор вместе** к совпавшей контрольной
точке, затем диапазоны читаются повторно. Хранятся 16 последних точек; если общая
точка не найдена, сервис останавливается с `registry_reorg`, без тихого сброса всей
истории. Если RPC отстаёт от сохранённого finalized-курсора, БД не откатывается
только из-за отставания ноды.

На один поток действует PostgreSQL advisory lock: одновременно пишет один
синхронизатор, даже если запущены несколько экземпляров приложения. Блокировка
не привязана к HTTP-сессии пользователя. При отмене ожидания соединение удаляется
из пула, чтобы не оставить на нём захваченную блокировку.

Первичная загрузка всё ещё требует RPC/CU и может занять несколько запусков по
5 минут. Теперь её прогресс сохраняется. Увеличивать начальный блок ради скорости
нельзя: ранние регистрации будут пропущены. Изменение start block или версии
декодера не применяется поверх старого курсора автоматически.

## Проверка состояния

```sql
SELECT chain_id, platform, encode(registry_address, 'hex') AS controller,
       start_block, last_scanned_block, encode(last_scanned_hash, 'hex') AS block_hash,
       updated_at
FROM registry_sync_state;

SELECT event_kind, count(*) FROM registry_events GROUP BY event_kind;

SELECT pg_size_pretty(pg_total_relation_size('registry_events')) AS history_size;
```

Логи дополнены этапами `registry_lock`, `registry_checkpoints`,
`registry_finalized`, `registry_tail` и полями `registry_saved_block`,
`registry_target_block`. Ошибка хранилища/невосстановимый checkpoint даёт HTTP 503;
таймаут — 504. Сырые SQL-ошибки и пароль подключения не выводятся.

## Тесты с настоящим Postgres

Обычный `go test` не использует mainnet и не подключается к `DATABASE_URL`.
Интеграционные тесты включаются **отдельной** переменной:

```powershell
$env:STAKEDAO_TEST_DATABASE_URL = "postgres://abievm:change_me_local@127.0.0.1:5432/abievm?sslmode=disable"
go test -mod=readonly -race -p 3 ./...
```

Используйте тестовую БД. Тесты создают случайную отдельную schema и удаляют только
её. Проверяются up/down, атомарность при SQL-сбое, сохранность курсора между
соединениями, блокировки/отмена и разделение chain ID. Сервисные тесты проверяют
инкрементальную загрузку, пустые диапазоны, порядок параллельных результатов,
продолжение после ошибки, актуальность временного хвоста и откат реорганизации.

`000001_registry.down.sql` **удаляет сохранённые регистрации и прогресс**. Это
ручной разрушительный rollback, не команда остановки приложения. Обычный
`docker compose -f compose.postgres.yml down` сохраняет named volume; `down -v`
его удаляет и не нужен для повседневного запуска.
