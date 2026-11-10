# Ручные ценовые маршруты и причины отсутствия APR

Маршруты для Ethereum weETH, frxETH, BOLD и трёх LP описаны в
[pinnedRoutes.go](pinnedRoutes.go). После проверки кэша сервис сразу выбирает
известный источник. Маршруты содержат адреса и правила чтения, а не цены.
БД и Redis не подключаются.

## Источники и единицы

| Токен | Контракт источника | USD-маршрут |
| --- | --- | --- |
| weETH `0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee` | Chainlink `0x5c9C449BbC9a6075A2c061dF312a35fd1E05fF22` | `weETH/ETH × ETH/USD` |
| frxETH `0x5E8422345238F34275888049021821E8E08CAa1f` | Curve WETH/frxETH `0x9c3b46c0ceb5b9e304fcd6d88fc50f7dd24b31bc` | `price_oracle() / 1e18 × WETH/USD` |
| BOLD `0x6440f144b7e50D6a8439336510312d2F54beB01D` | Curve BOLD/USDC `0xefc6516323fbd28e80b85a497b65a86243a54b3e` | `USDC/USD ÷ (price_oracle(0) / 1e18)` |

Chainlink weETH/ETH опубликован в
[официальном address book](https://github.com/bgd-labs/aave-address-book/blob/d81e02d3ed823a3f77909d03762c619e52ef6037/src/ChainlinkEthereum.sol).
На блокчейне подтверждены `description="weETH / ETH"`, decimals=18 и корректный
`latestRoundData`. Ответ — ETH за один weETH. ETH/USD берётся из существующего
feed `0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419`, decimals=8.
Изученный Aave CAPO не используется: он включает ограничение роста цены залога.

У frxETH-пула coin 0 — WETH, coin 1 — frxETH: EMA возвращает WETH за frxETH.
Проверены интерфейсы и формула публичного
[Plain2BasicEMA](https://github.com/curvefi/curve-factory/blob/5272fd2b1199f185d9cd94822be9262c387dc2c2/contracts/implementations/plain-2/Plain2BasicEMA.vy).
Цена frxETH не приравнивается ETH.

У BOLD-пула coin 0 — BOLD, coin 1 — USDC: EMA возвращает BOLD за USDC,
поэтому нужна **инверсия**. Проверены
[исходник NG](https://github.com/curvefi/stableswap-ng/blob/3332cd44656ec64b3c048885e1dd71955254b262/contracts/main/CurveStableSwapNG.vy)
и normalization rates `1e18` для BOLD / `1e30` для 6-decimal USDC.
USDC/USD читается через FeedRegistry, а не подставляется как `$1`.

EMA NG может выражать цену в нормализованных xp-единицах. Например,
weETH/WETH имеет динамический stored rate weETH около 1.1078, а EMA около 1.
Поэтому EMA этого пула не используется напрямую как weETH/ETH. Для weETH
выбран Chainlink, для BOLD/USDC допускаются только проверенные стандартные
normalization rates. Изменение этих условий оставляет цену недоступной.

EMA-маршруты — оценка рынка конкретного Curve-пула. Происхождение явно
обозначено `curve_ema_usd`; источник weETH — `chainlink_cross_usd`.

## Условия пригодности и кэш

- Все новые контрактные чтения привязаны к snapshot-блоку. В обычном режиме
  допускается пригодная кэш-котировка другого блока. Строгая диагностика
  использует отдельную map и для цен, и для зависимостей LP.
- Chainlink: ответ положительный, round/answeredInRound согласованы,
  timestamps валидны, возраст менее 25 часов. Cross-feed проверяет description.
- EMA: точные пары и порядок coins, положительная цена, период усреднения
  от 300 до 86400 секунд, наблюдение моложе двух часов без будущего timestamp.
- Timestamp цены — младшие 128 бит `ma_last_time`; у старого plain-пула
  верхняя упакованная часть отсутствует.
- На каждой стороне reference-пула требуется не менее $100000. Это политика
  маршрута, не гарантия защищённости EMA. Порог относится к глобальному
  reference-пулу, не к размеру StakeDAO vault.
- Deadline ограничивается сроком наблюдения/раунда и USD-anchor. TTL кэша
  остаётся 15 минут и не продлевает эти сроки.

Непригодный закреплённый источник не заменяется `$1`, `1 ETH`, spot-price или
другим непроверенным источником. Recursive LP и metapools остаются вне реализации.

Для weETH/WETH и BOLD/USDC закреплена ветка StableSwap NG, для frxETH/WETH —
MetaRegistry с известным pool-адресом. Существующие проверки пула и LP сохранены.
Тест одинаковых шести котировок дал **48 контрактных чтений с закреплёнными
LP-маршрутами против 55 с общим поиском**; цены совпали. Тёплая map не делает RPC.
Ручные маршруты сохраняются после перезапуска, поскольку находятся в коде.

Баланс, totalSupply, ERC4626 conversion, раунды и EMA-наблюдения обновляются.
Oracle-чтения оставлены прямыми `eth_call`: FeedRegistry может возвращать
access-controlled Chainlink aggregator, чей вызов через Multicall отличается
по `msg.sender`. Для USDC проверен именно реальный aggregator FeedRegistry.

## Mainnet-проверка

Блок **26385766**, timestamp `1794328799`: три пула, **105 RPC-попыток,
20.965 секунды**, без GetBlock CU и без ошибок полей.

| Vault | TVL USD | Rewards APR |
| --- | ---: | ---: |
| weETH/WETH | 1269279.79 | 1.195657% |
| BOLD/USDC | 389557.77 | 1.218926% |
| frxETH/WETH | 11.58 | 1.158407% |

Все три USD-цены отдельно пересчитаны из ответов источников на этом же блоке;
hash проверен повторно. Большие исходные количества сохранены строками:
[три пула](../../../examples/stakedao-oracle-routes/3pools.json),
[проверка источников](../../../examples/stakedao-oracle-routes/source-verification.json).

105 запросов включают проверки пригодности оракулов и расширенную диагностику;
это не прямое сравнение скорости с прежним прогоном без цен. Полный реестр
после правки не загружался. Другие пулы используют эти token-маршруты, если
их LP-модель и остальные компоненты уже поддерживаются.

## Как проверить причины APR

В подробном пуле появился `aprReasons`. Краткий основной ответ остаётся
из шести полей. Причины отражают предпосылки расчёта, не спрос на ликвидность.

| Причина | Условие |
| --- | --- |
| `empty_vault` | Прочитанное количество активов равно нулю. |
| `asset_price_missing` | Активы есть, но их USD TVL определить не удалось. |
| `reward_data_missing` | Не хватает данных о ставках/размещении/комиссиях; точное место — в errors. |
| `unsupported_model` | Модель протокола или распределения ещё не поддерживается. |

Дополнительно есть `asset_data_missing`, `reward_price_missing`,
`accounting_mismatch`: отсутствие количества активов, цены активной награды
и дефицит покрытия учтённых активов. У подтверждённого доступного APR, включая
`0%`, причин нет. Частичный APR может иметь причины. Пустой vault получает
`empty_vault`, а дополнительные проблемы сохраняются в errors. У непустого
vault может быть несколько причин.

В Swagger теперь можно указать любые 1–5 gauge:

```text
GET /reservesStakedao/ethereum/diagnostics?gauges=0x07a01471fA544D9C6531B631E6A96A79a9AD05E9&strictPrices=true
```

Это BOLD/USDC. Нужен **gauge**, не `address` краткого ответа (тот является vault).
При `gauges` количество определяется списком, `limit` игнорируется. Без списка
сохраняется стандартная выборка `limit=2/5`. Действуют `block`, `strictPrices`,
отсутствие logs/БД, ограничение 5 RPC/с и 300 попыток на запрос.

CLI для трёх новых маршрутов:

```powershell
$env:STAKEDAO_ETHEREUM_RPC_URL = "https://ethereum-rpc.publicnode.com"
go run -mod=readonly ./cmd/stakedao-check -diagnostics -limit 3 -gauges 0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2,0x07a01471fA544D9C6531B631E6A96A79a9AD05E9,0x4E21418095d32d15c6e2B96A9910772613A50d50 -strict-prices -output oracle-pools.json
```

Для всех пулов причины есть в `GET /reservesStakedao/ethereum?details=true`,
но этот запрос снова загружает полный реестр. Для отдельной причины достаточно
ограниченной диагностики выше.

Реальные примеры:

- [Пустой sdFXS/FXS](../../../examples/stakedao-oracle-routes/empty-vault.json):
  gauge `0x0060e266c2af65bfc4fd51b04c93d952df805630`, `aprReasons=["empty_vault"]`.
- [avUSD/evaUSDC без цены](../../../examples/stakedao-oracle-routes/missing-price.json):
  gauge `0x3285987bbd0b695ac48dc8571f519065b50aba07`, `aprReasons=["asset_price_missing"]`.
- [StakeDAO Balancer sdBAL](../../../examples/stakedao-oracle-routes/unsupported-model.json):
  gauge `0xdc2df969ee5e66236b950f5c4c5f8abe62035df2`,
  `aprReasons=["asset_price_missing","unsupported_model"]`. Модель Balancer
  и её цена этой правкой не реализовывались.

`reward_data_missing` воспроизведён тестом endpoint с непрочитанным `strategy()`.
В errors сохраняются `code/stage/contract/method`. При общей транспортной ошибке
вместо полного JSON может вернуться HTTP-ошибка; данные не заменяются нулём.

Для сохранённого подробного JSON:

```powershell
$r = Get-Content -Raw -Encoding UTF8 oracle-pools.json | ConvertFrom-Json
$r.pools | Select-Object name, apr, aprStatus, aprReasons
$r.errors | Select-Object gaugeAddress, field, code, stage, contract, method
```

Проверены единицы cross-rate, инверсия BOLD и отклонение от `$1`, stale/future/zero
ответы, liquidity/window/normalization/coin guards, срок USD-anchor, packed EMA
timestamp, экономия lookup при неизменных ценах, причины APR и новый endpoint.
Пройдены `go test ./...`, race-проверка затронутых пакетов, `go vet ./...`,
`go build ./...`; Swagger перегенерирован.
