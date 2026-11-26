# BOLD/USDC: маршрут наград; weETH/WETH: модель комиссий

## Исторический APR подключён — 26 ноября 2026

Тот же Swagger endpoint теперь по умолчанию использует проверенную публикацию
для root `0xbeafd406e6d206e131a4a37f4fa5fdc6e895422dcb1ff391bc8a54ff6c2d0af8`
и кампаний 2593/2594. Дополнительных параметров не требуется:

```text
GET /reservesStakedao/ethereum/bold-campaigns?fromBlock=26450710&block=26456929
```

Все алгоритмы и компактная проверенная публикация находятся в `boldCampaigns.go`.
Публикация закреплена на исходном коммите и SHA256: сервер не скачивает
изменяемый `last_merkle.json`, не вызывает API StakeDAO и не принимает проценты
или URL от пользователя. Фактический onchain root должен совпасть с
зарегистрированным. Проверяются периоды, funded amounts, суммы получателей
и целочисленная арифметика. Доказательства 15 уникальных получателей проверяются
при первом чтении встроенного набора; неизменяемые данные затем переиспользуются
в памяти процесса. Историческая baseline/delta-атрибуция взята из проверенной
публикации; все Transfer долей vault заново не воспроизводятся. Неизвестный root
даёт `publication_not_registered_for_root`, без угадывания данных распределения.

Для каждой кампании читаются **три исторические точки**: начало, середина и
конец. Timestamp-поиск выбирает последний блок не позднее нужного времени
(проверяется следующий блок, допустимый разрыв до 120 секунд). На этом блоке
проверяются `vault.asset()` и LP decimals=18, читается `vault.totalAssets()`.
Общий сервис цен в отдельном snapshot-кэше определяет BOLD/USD и LP/USD:
BOLD — проверенный Curve EMA + USD-источник USDC, LP — NAV компонентов Curve NG.
Проверки ликвидности/свежести сохраняются. TVL = `totalAssets × LP_USD`.
Источники цен, адреса, блоки и hash возвращаются в каждой точке; исторические
хеши перепроверяются перед возвратом результата.

Капитал в USD*seconds и средняя цена награды оцениваются трапециями между
этими тремя точками. Для равномерного campaign rate:

```text
historicalApr = allocatedBOLD × averageHistoricalBOLD_USD × 31536000 × 100
                / estimatedIntegral(vaultTVL_USD × dt)
```

Это **историческая оценка**, а не точный интеграл всех изменений TVL/цен.
Между точками предполагается линейное изменение; резкие промежуточные депозиты,
выводы или изменения цен могут повлиять на точность. Это агрегат наград
получателей vault, не индивидуальная доходность одного кошелька.

В `campaigns[i]` теперь возвращаются:

- `allocation.publication`: подтверждённая сумма, число получателей, root,
  исходные URL и commit;
- `allocation.valuation`: модель, допущения, три точки, средний капитал,
  капитал в USD*seconds, средняя цена BOLD и стоимость наград;
- `historicalApr`: годовая оценка в процентах, десятичная строка.

Если цена, старое состояние или бюджет недоступны, подтверждённая публикация
сохраняется в JSON: `allocationStatus=published_allocation_verified`,
`valuation.status=unavailable`, `historicalApr=null`, причина — в `reasons`.
Нет подстановки $1 или текущего TVL. Успешный компонент имеет статус
`historical_estimate_available`. `currentApr` остаётся null; результаты
не складываются с gaugeExtra или обычным APR/APY сервиса.

Бюджет остаётся 300 RPC и 5 RPC/сек на оба источника; цены и поиск старых
блоков входят в него. Последние 32 попытки резервируются для контрольных
чтений hash. Нужен архивный state RPC (например, Alchemy); GetBlock может
остаться источником логов. Postgres и новые миграции не требуются.
Офлайн-тесты включают полный путь через настоящий ContractSource с локальным
RPC, проверку historical block arguments, независимую рациональную проверку
APR, неизвестный root, повреждённую публикацию и reorg. Публичный PublicNode
отдаёт заголовки этого периода, но отказал в архивном `eth_call`; реальные
mainnet значения APR в этой проверке не получены, личные ключи не использованы.

Для ручного переноса обновляются `boldCampaigns.go`, `internal/handler.go`
и три Swagger-файла `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.
CLI также переведён на этот режим (`cmd/stakedao-check/main.go`), но для
Swagger его обновление не требуется. Основной расчёт пулов resolver не вызывает.

## Найдена публикация для проверенного корня — 25 ноября 2026

Корень из пользовательского `responseBoldCapaings_1795623573048.json`
на блоке **26456929**:
`0xbeafd406e6d206e131a4a37f4fa5fdc6e895422dcb1ff391bc8a54ff6c2d0af8`.
Он точно совпадает с опубликованным деревом StakeDAO:

- [Merkle-файл 1795176071](https://github.com/stake-dao/merkl-toolkit/blob/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795176071/merkle.json).
- [Распределение кампаний 2593/2594 в раунде 1795111271](https://github.com/stake-dao/merkl-toolkit/blob/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795111271/distribution.json).
- [Окна и веса держателей BOLD/USDC vault](https://github.com/stake-dao/merkl-toolkit/blob/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795111271/gauges/0x543D3Ee7Cfa494E8ffCC18364D130C8eED4B90ad.json).

Все ссылки закреплены на коммите, а не на изменяемом `last_merkle.json`.
Нулевой onchain ipfsHash не мешает найти эту публикацию на GitHub.
Индекс `data/distribution.json` связывает раунд 1795176071 с блоком 26456172,
предшествующим пользовательскому snapshot. Сопоставление с onchain root основано
на пользовательском проверенном ответе; нового RPC-чтения в этом исследовании нет.

**Независимая проверка данных:**

- Из всех листьев восстановлено полное дерево с исходным порядком листьев и
  сортировкой пар. Результат точно равен указанному root.
- Проверены все **2731** доказательство для **1162** аккаунтов, включая
  **31** BOLD-лист. Также восстановлены деревья до и после BOLD-раунда.
- Для каждого из **15** получателей прирост накопленного BOLD между раундами
  1795089695 и 1795111271 точно равен сумме опубликованных выплат двух кампаний.
  До целевого root 1795176071 все BOLD cumulative amounts остались неизменны;
  доказательства при этом изменились вместе с глобальным деревом.
- Периоды, токен, gauge и funded amounts согласованы с ранее прочитанными
  onchain incentives. Проверены опубликованные веса, rate и арифметика выплат.

| ID | Получателей | Выделено в кампанию, BOLD | Распределено, BOLD | Остаток от целочисленного rate, raw |
| --- | ---: | ---: | ---: | ---: |
| 2593 | 5 | 10.162926236299420784 | 10.162926236299411596 | 9188 |
| 2594 | 15 | 349.080457011333255452 | 349.080457011333212616 | 42836 |

Всего получателям начислено **359.243383247632624212 BOLD**. Разница с суммой
финансирования — **52024 raw**, то есть **0.000000000000052024 BOLD**.
Она в точности объясняется `floor(amount/duration) * duration`, не новой
процентной комиссией. Это отдельно от ранее наблюдавшегося возврата Router.

Опубликованный алгоритм использует веса
`weight(user) = integral(balance(user,t)/totalSupply(t) * dt)` с масштабом 1e36,
затем делит сумму пропорционально весам. Остаток от деления между получателями
достаётся последнему получателю в порядке распределения. Исходники:
[TWAB](https://github.com/stake-dao/merkl-toolkit/blob/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/src/utils/twab.ts),
[rate и распределение](https://github.com/stake-dao/merkl-toolkit/blob/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/src/2_distribution.ts),
[накопление entitlement и дерево](https://github.com/stake-dao/merkl-toolkit/blob/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/src/utils/merkle.ts).

Суммы идут в накопленные Merkle-права держателей, а не в прямой BOLD claim
самого vault (его адреса среди BOLD claim accounts этого root нет). Источник
этих incentive в реестре StakeDAO — `gauge`: downstream Merkle-выплату нельзя
автоматически прибавлять к upstream gaugeExtra как независимую награду.

**Граница достоверности:** доказательства подтверждают авторизованные суммы,
но не справедливость расчёта eligibility. Исторические Transfer vault нами
независимо не воспроизведены; сверена арифметика опубликованных весов.
Эти веса не являются интегралом USD-капитала. Для годовой исторической оценки
нужны ещё пригодные исторические цены и капитал соответствующего периода.
Завершённые кампании не определяют текущий forward APR.

Сохранены ссылки, SHA256 исходников, суммы и доказательства прироста получателей:
[stakedao-bold-published-root-verification.json](../../../examples/stakedao-bold-published-root-verification.json).
Поиск источника для этих двух кампаний завершён. Runtime и
`BOLDAllocationResolver` в этом шаге не менялись; автоматический API/RPC
fallback не добавлялся.

## Изолированный модуль BOLD — 24 ноября 2026

В [boldCampaigns.go](boldCampaigns.go) добавлен `BOLDCampaignInspector`.
Он не подключён к загрузке пулов StakeDAOService, основному APR/APY или Postgres.
Для явного запуска есть отдельный исследовательский endpoint в Swagger.
Все новые рабочие типы и функции находятся в одном файле; используются уже
существующие ABI/адреса и типы Multicall из пакета stakedao.

```go
// stateClient и logsClient — уже открытые *ethclient.Client.
// При необходимости это Alchemy для state и GetBlock для logs.
// Проверенная публикация и исторические цены/капитал.
inspector := stakedao.NewBOLDPublishedCampaignInspector(stateClient, logsClient)
report, err := inspector.Inspect(ctx, 26447429, 26456929)
```

Окно в примере историческое и содержит ранее исследованную транзакцию.
Для него RPC должен разрешать архивные чтения.

### Запуск через Swagger — 25 ноября 2026

Обычный сервер приложения предоставляет отдельный маршрут:

```text
GET /reservesStakedao/ethereum/bold-campaigns?fromBlock=26450710&block=26456929
```

[Открыть метод в Swagger](http://localhost:8081/swagger/index.html#/StakeDao/getStakeDaoBOLDCampaigns).
В разделе StakeDao выбрать «Проверить кампании дополнительных наград BOLD/USDC»,
нажать Try it out и Execute. Параметры обязательные; в форме предзаполнен
контрольный исторический диапазон с кампаниями 2593/2594. `fromBlock` и `block`
включительны, `block` также задаёт snapshot всех чтений состояния. Gauge не
нужно вводить: этот метод относится только к проверенной паре BOLD/USDC.

JSON ответа — сам `BOLDCampaignReport`, с кампаниями, fundingStatus, основаниями
недоступности и nullable APR. Его можно скачать из Swagger. При ошибке ответ
содержит безопасные `error` и `reason`; отказ RPC не превращается в пустой
успешный отчёт. Неверные параметры дают 400, отсутствие основного RPC — 503,
отказ/несогласованность RPC — 502, таймаут — 504.

Используются `STAKEDAO_ETHEREUM_RPC_URL` (Alchemy для состояния) и
`STAKEDAO_ETHEREUM_LOGS_RPC_URL` (GetBlock для событий) из существующей
конфигурации запуска **сервера**. Если второй URL не задан, оба вида чтения
выполняются основным RPC. Новая конфигурация GoLand/CLI, БД или миграция для
этой проверки не нужны. Запрос ограничен двумя минутами, 300 RPC и 5 RPC/сек,
оба клиента дополнительно участвуют в общем ограничителе RPC приложения.

Алгоритмы трёх задач остаются в `boldCampaigns.go`; handler только проверяет
параметры, вызывает их и возвращает JSON. Обновлять нужно этот файл,
`internal/handler.go`, `internal/server.go`, `docs/docs.go`, `docs/swagger.json`
и `docs/swagger.yaml`. Основной `/reservesStakedao/ethereum` не вызывает новую
проверку. Исторические/текущие APR остаются null при неподтверждённой allocation.

### Дополнительный вариант: запуск без HTTP-сервера

В существующей команде `cmd/stakedao-check/main.go` добавлен отдельный режим.
После копирования одного `boldCampaigns.go` нужно также обновить этот `main.go`;
никакие изменения `cmd/main.go`, HTTP handler, Swagger или миграции не нужны.

Из корня проекта:

```sh
go run ./cmd/stakedao-check -bold-campaigns -from 26450710 -block 26456929 -output bold-campaigns.json
```

В **этом терминале или этой конфигурации GoLand** должны быть заданы:

```env
STAKEDAO_ETHEREUM_RPC_URL=<полный URL Alchemy для исторического состояния>
STAKEDAO_ETHEREUM_LOGS_RPC_URL=<полный URL GetBlock для событий>
```

Если второй URL не задан, события читаются через основной RPC. Автоматического
переключения при отказе нет. Переменные старой конфигурации запуска сервера
сами по себе не наследуются новым терминалом/конфигурацией; `.env` автоматически
не загружается. Не вставляйте URL в аргументы команды и не присылайте ключи.

В GoLand можно создать **Go Build → Package** с package path
`ABIevm/cmd/stakedao-check`, рабочей папкой в корне проекта и Program arguments:

```text
-bold-campaigns -from 26450710 -block 26456929 -output bold-campaigns.json
```

Результат сохраняется после успешной проверки. Ошибка оставляет предыдущий JSON
без изменений и завершает программу с ненулевым кодом. Без `-output` JSON идёт
в stdout; сообщения о ходе проверки — в stderr. Таймаут запуска — 2 минуты.
Другие режимы/флаги stakedao-check в этой команде смешивать не нужно.

В этом историческом окне ранее найдены кампании **2593/2594**. Проверяем
`scanComplete:true`, `fundingStatus:verified_router_campaign_funding`, сумму,
период и соответствие блока. При недоступной истории RPC сначала устраняется
ошибка чтения, она не считается отсутствием кампаний. Нормальный результат
при resolver=nil: `allocation_rules_unverified`, `historicalApr:null`,
`currentApr:null`. Следующий этап — подтверждение модели распределения;
проверка финансирования не означает, что фактическая доля vault уже известна.

Офлайн-тесты (требуют обновлённые существующие файлы `_test.go`):

```sh
go test ./cmd/stakedao-check -count=1
go test ./internal/service/stakedao -run TestBOLD -count=1
```

Они используют локальные RPC-заглушки и сохранённые payloads; CU провайдеров
не расходуют. Если скопирован только рабочий `.go`, этих новых тестов в локальном
проекте нет; сообщение `[no tests to run]` не подтверждает проверку модуля.

1. **Поиск и проверка кампаний.** Сканируется только явно заданное окно
   создания кампаний, события читаются диапазонами до 1500 блоков. Проверяются
   chain ID и hash границ обоих источников, Controller, Receiver и Router.
   `IncentiveAdded` сверяется с `incentives(id)` на конечном блоке, включая
   fromChainId, sender/manager, токен и gauge. Прочитать последние восемь ID
   недостаточно. Успешное пустое окно не означает отсутствия всех наград.
2. **Проверка финансирования и граница правил распределения.** По receipt
   сверяются Router pull и переводы BOLD Receiver → Router → AllMight →
   distributor, сумма кампаний и возврат в Receiver. Проверенная сумма не
   объявляется комиссией или выплатой держателю vault. Считываются активные
   `root()` и `ipfsHash()`. `VerifyBOLDClaimProof` повторяет double-keccak leaf и
   sorted-pair proof из проверенного UniversalRewardDistributor; это проверка
   накопленного права account/token, без привязки к конкретной кампании/пулу.
   Остаток к получению требует отдельного чтения `claimed(account,reward)`.
3. **Отдельная историческая оценка.** `CalculateBOLDHistoricalAPR` принимает
   чистую сумму BOLD, подтверждённую для данного vault и периода, USD-цену и
   интеграл капитала за весь период. Формула:
   `netBOLD × BOLD_USD × 31536000 × 100 / integral(vaultCapitalUSD × dt)`.
   Проверяются ID, vault, snapshot hash, начало/конец периода, лимит суммы
   финансирования и ссылки на основания. Незавершённый период не превращается
   в историческую ставку. Оценки кампаний автоматически не складываются.

**Правила распределения не придуманы.** Действительная привязка выплат и
интеграл капитала должны быть получены будущей проверенной реализацией
`BOLDAllocationResolver`. Это доверенная граница источника данных, а не флаг
`verified=true`: локальная арифметика не доказывает истинность входных данных.
При resolver=nil каждая найденная кампания имеет `historicalApr:null` и
`allocation_rules_unverified`, даже если финансирование полностью сверено.
`currentApr` остаётся null и при наличии исторических оценок. Это не готовый
расчёт текущего APR дополнительных наград и не реализация Angle Merkl.

Ограничения встроенного чтения: 160 диапазонов, 64 кампании, 32 receipt,
20000 событий, 300 RPC, 5 RPC/сек на оба клиента суммарно в одном запуске.
Ошибка диапазона, реорганизация, другой маршрут, неравенство финансирования
или неоднозначный multi-pull завершают проверку ошибкой. Встроенные ограничения
не распространяются автоматически на будущие сетевые запросы resolver.
RPC URL и исходные тексты ошибок провайдера в результат не копируются.

Тесты добавлены в существующий `StkDaoSRV_test.go`: сохранённые payloads реальной
транзакции и getters кампаний 2593/2594, повреждённые переводы/дубликаты,
неполная история, отказ RPC, изменение маршрута/hash, отсутствие allocation,
проверка Merkle leaf и единиц исторического APR. Это офлайн-проверка, не новая
загрузка mainnet и не подтверждение реальных правил распределения.

## Что проверено и что изменено

Это результат read-only исследования, а не новый алгоритм APR в runtime.
Проверены getters, верифицированные исходники и receipt реальной транзакции.
Цифра gauge-extra не прибавлялась к APR vault; weETH APY не заполнялась
непроверенной оценкой. Curve/AAVE не менялись.

Последующая реализация: [отдельная оценочная модель weETH и BOLD route diagnostics](WEETH_FEE_MODEL_RU.md).

Все чтения текущего состояния выполнены на Ethereum block **26456929**,
timestamp **1795185215**, hash
`0xd2da74c4e0a32e8f7cb17f967333aa00c77abfa3827e3e59ef7b0fe3e643c892`.
Hash проверен повторно. Это новый срез, не блок 26450201 из ответа пользователя.
Источник RPC — PublicNode; личные GetBlock/Alchemy ключи не использовались.

Машиночитаемые данные и проверенные суммы:
[stakedao-bold-route-audit.json](../../../examples/stakedao-bold-route-audit.json).

## Контракты BOLD/USDC

| Роль | Адрес | Чтение / проверка |
| --- | --- | --- |
| Curve gauge G | `0x07a01471fa544d9c6531b631e6a96a79a9ad05e9` | rewards_receiver(L), claimable_reward(L,BOLD), reward_data(BOLD) |
| StakeDAO vault V | `0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad` | gauge(), asset(), strategy(), getRewardTokens(), isRewardToken(BOLD), getRewardsDistributor(BOLD), rewardData(BOLD) |
| Locker L | `0x52f541764e6e90eebc5c21ff570de0e2d63766b6` | Позиция в G; получатель задаётся через G.rewards_receiver(L) |
| Новый Receiver R | `0xaee4d726c1ccdb82e79e7cf9922fa6f5b89320d3` | rewardVault(), rewardRouter(), legacyRewardReceiver() |
| Legacy Receiver R0 | `0x8e980f6d681a02a701d4ce5ef68f4786351a3d8f` | rewardVault(); он является distributor BOLD в V |
| RewardRouter | `0x9a207a85e372fcdac3014f945a65868f2a05ba12` | PROTOCOL_CONTROLLER(), merkleDistributor() |
| AllMight | `0xdbd24b092f686b12650ec1450e3a7138f714506c` | Верифицированный исполнитель weiroll-планов; не ABI CampaignRewardsDistributor |
| CampaignRewardsDistributor D | `0xd4898a378ea555595c4e7dbde722b134a3f346d1` | incentives(id), nbIncentives(); IncentiveAdded; claim(account,reward,claimable,proof) |
| BOLD | `0x6440f144b7e50d6a8439336510312d2f54beb01d` | balanceOf(address); Transfer |

Подтверждены G.rewards_receiver(L)=R, R.rewardVault()=V,
R.legacyRewardReceiver()=R0, R0.rewardVault()=V и Router.PROTOCOL_CONTROLLER()
совпадает с текущим Controller. BOLD зарегистрирован в V; distributor BOLD=R0.

### Два разных пути

Из исходников Receiver и Router:

```text
путь vault: G → R → R0 → V.depositRewards(BOLD, amount)
путь Merkle: G → R.pullRewards(BOLD) → Router → AllMight → D
```

Новый R с настроенным Router допускает pull/distribution только через Router.
Router.execute(bytes[]) проверяет разрешение Controller.allowed и выбирает
подоперацию pullToMerkle, distribute или distributeAll. Наличие адреса
merkleDistributor само по себе ещё не доказывает выбранный путь.

При пути vault R сверяет getRewardsDistributor: если им является R0, переводит
токен в R0 и вызывает его distributeRewardToken. R0 проверяет регистрацию,
distributor и окончание предыдущего периода перед V.depositRewards.

На блоке проверки V.rewardData(BOLD): lastUpdate=0, periodFinish=0,
rewardRate=0, rewardPerTokenStored=0. Ветка линейных выплат V не запускалась.
Балансы BOLD: R=2.275469070685196406, R0=29.768457214399919866, V=0.
G.claimable_reward(L,BOLD)=554.397917650267476258 — накопленная сумма,
не ставка выплаты. Её нельзя произвольно делить на неделю для получения APR.

## Реальный путь через Merkle подтверждён транзакцией

[Транзакция 0x4afddda51180de64645535501deb9b554c44bba0a9e543b40f3564c5f6f76236](https://etherscan.io/tx/0x4afddda51180de64645535501deb9b554c44bba0a9e543b40f3564c5f6f76236),
block **26450710**, receipt status=1. Она произошла **после** блока 26450201
из пользовательского JSON. Не переносим её последствия назад во времени.

| Перевод BOLD в receipt | Количество |
| --- | ---: |
| Curve gauge → R | 361.330126695910522636 |
| Convex StashTokenWrapper → Sidecar → R | 0.029442879220323793 |
| R → Router → AllMight | 361.518852318317872642 |
| AllMight → D, incentive 2593 | 10.162926236299420784 |
| AllMight → D, incentive 2594 | 349.080457011333255452 |
| AllMight → R | 2.275469070685196406 |

Router испустил RewardsPulledToMerkle с точными G и BOLD. D испустил
IncentiveAdded с теми же адресами для ID 2593/2594. Getters incentives(2593)
и incentives(2594) независимо подтвердили поля событий, sender/manager=AllMight,
fromChainId=1. Остаток, возвращённый в R, не называем комиссией без доказательства.
Сумма финансирования кампаний и возврата точно равна pulled-сумме.
Источник малого Convex-перевода — StashTokenWrapper
`0x1d4bbca365522a0e79e91fdc92b202dc7ce29bb3`, не сам reward pool. Его
верифицированный transfer пропускает перевод underlying токена только при
вызове настроенным rewardPool. В receipt за ним следует перевод Sidecar в R.
Pulled-сумма включает ранее находившиеся в R токены, поэтому не равна только
новым переводам gauge+Sidecar в этой транзакции.

| ID | start (unix) | end (unix) | duration, сек |
| --- | ---: | ---: | ---: |
| 2593 | 1795023887 | 1795046483 | 22596 |
| 2594 | 1795046543 | 1795110167 | 63624 |

Периоды ретроспективные и закончились до блока исследования. Поэтому
amount/duration — характеристика финансируемого прошлого периода, не готовый
текущий APR. На snapshot nbIncentives=2595. Факт существования этих кампаний
не доказывает текущую ставку, долю конкретного vault или claimable пользователя.

Здесь подтверждено **Merkle-распределение StakeDAO**. Из названия функции Router
нельзя заключить, что использован сервис Angle Merkl или его API. Для полной
оценки нужно установить правила разбиения финансирования между позициями,
данные опубликованного root/proof, периоды позиций и исключение повторного учёта.
Один поток gauge может впоследствии финансировать кампанию: gaugeExtra и её
оценка не являются двумя независимыми источниками дохода.

### Ограничение поиска событий

Запрошен диапазон 26406929–26456929 интервалами по 1500, четыре worker,
общий запуск не чаще 4 RPC/сек. 27 из 34 диапазонов провайдер отклонил как
архивные. Успешно прочитан непрерывный хвост 26447429–26456929, в нём найдено
вышеуказанное событие. **Это не полный реестр и не доказательство отсутствия
ранних событий или иных маршрутов.** Receipt и прямые getters получены успешно.

## weETH/WETH: что есть в приложении

Проверен [curveService.go](../curveService.go); производственный код не изменён.

| Функция | Принцип, который можно повторно использовать |
| --- | --- |
| load24hVolumeUSD | Разбиение диапазона и ограниченное число worker |
| processVolumeLogsChunk | eth_getLogs, выбор пары, привязка обмена к pool |
| decodeCurveSwapEvent / loadCurveSwapABI | Типизированный разбор TokenExchange; его поля совпадают с проверенным NG-пулом |
| getPoolFeeARP | Идея нормализации дохода от комиссий на TVL; функция возвращает APR |

Текущий getPoolFeeARP по существу считает
`volume24h × currentFee × 365 / currentTVL × 100`.
Это приближение по текущим параметрам. Оно не отделяет admin-долю и не
восстанавливает фактическую динамическую комиссию каждой прошлой операции.
Следовательно готовой точной модели weETH или разложения staking/trading
в существующем коде нет. Надпись APY в консоли не меняет формулу APR.

Не нужно копировать глобальные map/BlockchainClientS, context.Background,
эвристическое переопределение decimals, молчаливое исключение крупных swaps
и замену ошибки отсутствующей истории нулём. Новый слой должен использовать
контекст/RPS/cache и проверки snapshot уже существующего StakeDAO-сервиса.

## Проверенный контракт weETH/WETH

- Pool / LP: `0xdb74dfdd3bb46be8ce6c33dc9d82777bcfc3ded5`.
- Gauge для Swagger: `0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2`.
- Factory.get_pool_asset_types(pool) = [0,1]: standard WETH и oracle-rate weETH.
- stored_rates() = [1000000000000000000,1108570196779355729].
- fee() = 500000 / 1e10 = **0.005%** (базовая комиссия).
- admin_fee() = 5000000000 / 1e10 = **50%** комиссии протоколу.
- offpeg_fee_multiplier() = 100000000000 / 1e10 = 10.

Последнее означает множитель механизма увеличения комиссии при дисбалансе,
не фиксированную комиссию 10%. Фактическая комиссия swap вычисляется в
__exchange через _dynamic_fee на средних нормализованных балансах входа/выхода.
Простой вызов dynamic_fee в конце блока не восстанавливает её для каждой
транзакции, особенно при нескольких swaps в одном блоке.

stored_rates weETH меняется и участвует в xp/invariant. Поэтому два snapshot
virtual price смешивают fee growth с изменением weETH-rate. Вычесть полный
процент weETH также нельзя: LP содержит разные доли WETH/weETH во времени,
на них влияют swaps и ввод/вывод капитала.

## Как строить модель

Практический путь из существующего кода — событийная история **фактических
комиссий**, отдельно от изменения обменного курса weETH:

1. Читать проверенные события конкретного NG-пула с cursor, сортировать по
   block/tx/log index, исключать дубликаты и removed logs, учитывать reorg.
   Не сканировать заново 30 дней при каждом HTTP-запросе.
2. Восстанавливать gross fee и admin fee в исходных единицах bought token из
   фактического состояния операции (replay/подходящие traces). TokenExchange
   содержит net tokens_bought, но **не содержит готовую fee-сумму**. Текущую
   fee или post-block dynamic_fee можно использовать лишь в явно обозначенной
   оценке, не выдавать её за точный исторический расчёт.
3. Считать retained LP fee = gross fee − admin fee; для согласованности с
   существующей fixed-rate моделью отдельно учитывать комиссии операций
   ликвидности, а не только swaps. Не учитывать изменение oracle rate как fee.
4. Приводить фактические fee-суммы к цене и капиталу того же периода. Возврат
   за интервал — feeValue / capital соответствующей LP-позиции; произвольно
   делить все месячные комиссии на сегодняшний TVL нельзя. Приближение по
   дневному среднему TVL допустимо только с явной маркировкой оценки.
5. По проверенным интервальным fee returns r_k получить R=product(1+r_k)-1,
   затем annualized APY=(1+R)^(31536000/elapsedSeconds)-1. Сохранить окна 7/30,
   основной период 30, полное покрытие и причины недоступности. Простая
   годовая сумма комиссий / TVL — fee APR, не эта APY.

Идея для независимой сверки: admin_balances(i) отражает накопленные комиссии
протокола в токенах. С учётом точных изъятий admin fees и постоянной доли 50%
можно сверять накопление LP-комиссий с admin-учётом, учитывая округления.
Это **кандидат на проверку**, не реализованный или доказанный универсальный
метод: нужно отличать admin-выводы от обычных выплат пула, учитывать все операции
и правило округления конкретной реализации. Одних двух admin_balances недостаточно.

Историю обменного курса weETH можно затем показывать отдельным компонентом.
Её нельзя просто складывать с fee APY и называть точным итогом LP: остаются
перебалансировки и рыночная оценка. Если нет пригодных исторических данных,
trading остаётся unavailable; приближение не подменяет эту проверку.

## Подтверждение предыдущей реализации через Alchemy

В предоставленном пользователем response_1795184555467.json, block 26450201,
оба fee-window у BOLD/USDC и frxETH/WETH доступны. Четыре APY независимо
пересчитаны из raw VP и elapsed seconds; результаты совпали:

| Pool | 7-day annualized APY | 30-day annualized APY |
| --- | ---: | ---: |
| BOLD/USDC | 1.0215660995691263% | 0.8390537761671611% |
| frxETH/WETH | 0.1554327762317654% | 0.11741925529194704% |

Это проверка арифметики предоставленных данных, не повторный запуск RPC с
личным Alchemy ключом. В примере сохранены границы, VP и источники этого среза.
Она не добавляет поддержку weETH или точного Merkle APR.

## Исходники

- [RewardReceiver](https://etherscan.io/address/0xaee4d726c1ccdb82e79e7cf9922fa6f5b89320d3#code).
- [Legacy Receiver](https://etherscan.io/address/0x8e980f6d681a02a701d4ce5ef68f4786351a3d8f#code).
- [RewardRouter](https://etherscan.io/address/0x9a207a85e372fcdac3014f945a65868f2a05ba12#code).
- [AllMight](https://etherscan.io/address/0xdbd24b092f686b12650ec1450e3a7138f714506c#code).
- [Convex StashTokenWrapper](https://etherscan.io/address/0x1d4bbca365522a0e79e91fdc92b202dc7ce29bb3#code).
- [CampaignRewardsDistributor](https://etherscan.io/address/0xd4898a378ea555595c4e7dbde722b134a3f346d1#code).
- [Curve gauge V6](https://etherscan.io/address/0x07a01471fa544d9c6531b631e6a96a79a9ad05e9#code).
- [weETH/WETH StableSwap NG](https://etherscan.io/address/0xdb74dfdd3bb46be8ce6c33dc9d82777bcfc3ded5#code).
