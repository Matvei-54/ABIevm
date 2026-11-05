# Проверка пяти пулов StakeDAO

Добавлен ограниченный режим диагностики двух или пяти заранее выбранных пулов.
Он использует текущие регистрации Controller, общий snapshot-блок, не сканирует
историю и не обращается к PostgreSQL.

## Запуск

В Swagger выберите `GET /reservesStakedao/{network}/diagnostics`, сеть
`ethereum`, параметр **limit=5**:

```text
http://localhost:8081/reservesStakedao/ethereum/diagnostics?limit=5
```

Без параметра остаются два исходных пула. Другие значения возвращают 400
до обращения к RPC. CLI из папки с `go.mod`:

```powershell
go run -mod=readonly ./cmd/stakedao-check -diagnostics -limit 5 -output stakedao-validation-5.json
```

RPC берётся из существующей `STAKEDAO_ETHEREUM_RPC_URL`.
Для отдельной проверки можно использовать PublicNode:

```powershell
$env:STAKEDAO_ETHEREUM_RPC_URL = "https://ethereum-rpc.publicnode.com"
go run -mod=readonly ./cmd/stakedao-check -diagnostics -limit 5 -output stakedao-validation-5.json
```

Результат Swagger можно сохранить:

```powershell
Invoke-RestMethod "http://localhost:8081/reservesStakedao/ethereum/diagnostics?limit=5" |
  ConvertTo-Json -Depth 30 |
  Set-Content -Encoding UTF8 stakedao-validation-5.json
```

Лимиты: **5 RPC/с, 300 попыток, 5 минут** на запрос. `-diagnostics` требует
`-limit 2` или `-limit 5` и не совмещается с `-gauges/-from/-to`.
Обычная команда для произвольных 1–5 gauge сохранена. Файл CLI записывается
только после успешного завершения запроса.

## Выбранные пулы и данные диагностики

| Пул | Gauge | Что проверяем |
| --- | --- | --- |
| crvUSD/USDC | `0x95f00391cb5eebcd190eb58728b4ce23dbfa6ac1` | MetaRegistry NAV, Locker + Convex Sidecar. |
| FXS/ETH | `0x009acd89535dabc270c93f9b39d3232105fef453` | MetaRegistry NAV, излишек 0.003 LP, нулевой текущий CRV-поток. |
| evaUSDC/evaUSDT | `0xbe6228125448e40f5dcbb2dbdf7ff3f5721000b4` | StableSwap NG NAV, конверсия обоих ERC4626-компонентов, Sidecar. |
| frxUSD/sUSDS | `0x52618c40ddba3cbbb69f3aaa4cb26ae649844b17` | StableSwap NG NAV, ERC4626 sUSDS, Sidecar. |
| crvUSD/USDT | `0x4e6bb6b7447b7b2aa268c16ab87f4bb48bf57939` | MetaRegistry NAV, компоненты с разными decimals, Sidecar. |

Это пять LP-vault. ERC4626 проверяется на **компонентах LP**, а не через
предположение, что сам LP является ERC4626. Ненулевые активы подтверждены
текущим чтением; список адресов сам по себе этого не гарантирует.

В `pools[].diagnostics` находятся:

- `primary`: inflation/relative weight/working balance/supply Locker,
  его gross CRV rate, gross rate Sidecar, доли и суммы комиссий Accountant,
  итоговый net rate. Ставки — в reward tokens/second, доли комиссий — от 0 до 1.
- `convex`: проверенные связи pid/LP/gauge/BaseRewardPool, исходные rate/supply/period,
  результат PoolUtilities и сравнение CRV-ставки. Для Sidecar также его баланс,
  собственная ставка и разница округления.
- `prices`: использованные USD-котировки, источник, контракт, блок, время,
  срок годности; дополнительно — доступные сведения о компонентах из кэша.

Цена, использованная в TVL/APR, удерживается в отчёте, даже если другой запрос
обновит общую map. Кэш проверяется первым. В повторном запросе цена может
принадлежать более раннему допустимому блоку: это видно по
`prices[].blockNumber/updatedAt`. В сохранённом примере все цены получены
на блоке результата. Сведения о компонентах читаются из текущего кэша и также
имеют собственные метки блока/времени. Для старых/custom-источников происхождение
может быть неизвестно (block=0). Дополнительных RPC за ценой для отчёта нет.

Контрольная формула:

```text
helper CRV raw = floor(active BaseRewardPool.rewardRate × 10^18 / totalSupply)
own CRV/sec = helper raw × sidecarBalanceRaw / 10^36
```

Последняя формула использует 18 decimals CRV. Helper округляет раньше нашего
расчёта; допустимая разница ограничена `sidecarBalanceRaw / 10^36` CRV/sec.
CVX mint projection и actual extra сохраняются раздельно даже при одинаковом
адресе токена. Контрольные ставки не добавляются повторно в APR.

Для FXS/ETH Convex pid=69 — **внешний ориентир**. У фактического размещения
StakeDAO нет Sidecar, поэтому own Sidecar rate в контроле null. Это не ошибка
и не основание добавлять Convex-доходность этому vault. Статус `matched`
проверяет CRV-формулу, а не весь APR/цену. При нулевом Convex supply статус
`zero_supply`, ошибке чтения — `unavailable`, расхождении — `mismatch`.
Эти статусы не меняют вычисленный APR и не заменяют неизвестную ставку нулём.

## Результат чтения и пересчёта

Ethereum, блок **26349407**, timestamp `1793891015`. Через PublicNode:
**35.9 секунды, 178 RPC-попыток**, без GetBlock CU. Все пять vault непустые,
`aprStatus=available`, ошибок полей нет. В логах есть 8 ожидаемых reverts
проб возможностей ценовых контрактов; они не стали ошибками расчёта.

Сохранены [полный ответ](../../../examples/stakedao-validation-5.json)
и [сравнение с опубликованным API](../../../examples/stakedao-validation-comparison.json).
Из сохранённых чисел отдельно пересчитаны TVL, Locker gross rate, комиссии,
net rate и APR каждого потока. Все пять контролей PoolUtilities имеют `matched`.

| Пул | Наш rewards APR | CRV APR API | Trading APY API | Разница CRV, п.п. |
| --- | ---: | ---: | ---: | ---: |
| crvUSD/USDC | 3.715848% | 3.806460% | 1.98% | −0.090612 |
| FXS/ETH | 0% | 0% | 0.86% | 0 |
| evaUSDC/evaUSDT | 9.889507% | 9.731370% | 0.01% | +0.158137 |
| frxUSD/sUSDS | 1.478396% | 1.612860% | 3.10% | −0.134464 |
| crvUSD/USDT | 4.403546% | 4.462937% | 2.41% | −0.059391 |

Источник — опубликованный
[Ethereum V2 JSON StakeDAO](https://github.com/stake-dao/api/blob/cc776d695be8c2d79e05c340866f6e1d6247c97d/api/strategies/v2/curve/1.json),
commit `cc776d695be8c2d79e05c340866f6e1d6247c97d`; SHA-256 исходного JSON
записан в сравнении. Live UI напрямую из этой среды недоступен. В JSON нет
snapshot-блока: **точное совпадение с UI на одном блоке не проверено**.

Наш APR включает текущие rewards, торговая APY не включается; `apy` остаётся null.
У API отдельно взяты CRV APR и Trading Fees APY, а не общий total.
Порядок токенов в имени API иногда обратный; соответствие проверено по
gauge и vault, а не по имени.

Разные котировки объясняют часть расхождений. В сравнении есть
`ourCrvAprUsingPublishedPricesPercent`: наш net CRV rate переоценён по ценам
LP и CRV API, при нашем количестве активов. Получаются примерно
3.754305%, 0%, 10.031809%, 1.494172%, 4.447324% соответственно.
**Одни цены не объясняют всю разницу**, особенно у frxUSD/sUSDS и eva-пула.
Не установлено, какая часть остатка связана с моментом данных, моделью
прогноза эмиссии UI, периодом распределения Convex или политикой комиссий API.
Оснований подгонять onchain-формулу под несинхронизированный показатель нет.

Для точного совпадения нужны данные UI/API с номером блока и методикой CRV APR,
затем сравнение со ставками Locker и фактическим 7-дневным потоком BaseRewardPool
на этом блоке. Текущий результат подтверждает арифметику и контроль Convex,
но не доказывает идентичность всей методики нашему UI-ориентиру.

## Область изменения и проверки

Краткий ответ основного endpoint остаётся из шести полей. В обычной загрузке
PoolUtilities не вызывается. `TokenPriceService.GetTokenQuote` возвращает
происхождение цены с прежними проверками TTL/oracle validity; числа цен и
порядок поиска не менялись. Redis, PostgreSQL, Curve/Aave сервисы и
recursivePools не менялись.

Проверены limit=2/5 и ошибки параметров, отсутствие сканирования истории,
границы периода, нулевой supply, несовпадение helper, ошибка чтения,
раздельные одинаковые токены разных источников, единицы ставок, комиссии,
сохранение использованной котировки при обновлении кэша и происхождение
NG/MetaRegistry/ERC4626-цен. Пройдены `go test ./...`, race-проверка затронутых
пакетов, `go vet ./...`, `go build ./...`; Swagger перегенерирован.
