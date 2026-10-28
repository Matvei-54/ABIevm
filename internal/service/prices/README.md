# Цены токенов: кэш и контракты

StakeDAO и `cmd/stakedao-check` используют общий `TokenPriceService` с
`ContractSource`. В этом workflow DeFiLlama/HTTP-источники не вызываются.
Старая Curve-реализация и `recursivePools` не менялись; её прежние HTTP-вызовы
не переведены на новый сервис в рамках этой задачи.

## Последовательность

1. Поиск свежей цены по `(chainID, tokenAddress)` в `prices.Store`.
2. Если цены нет/она истекла — чтение контрактов через `EnsurePricesAt`.
3. Сохранение цены токена и успешно оценённых компонентов в тот же Store.

Сейчас Store — `MemoryStore` с RWMutex. Для будущего Redis достаточно реализовать
`Get/Set` и сохранить поля Quote, включая `UpdatedAt` и `ValidUntil`. Redis пока
не подключён. `GetTokenPrice` по-прежнему читает только кэш: RPC запускается
явно через `EnsurePricesAt`, а не при каждом чтении цены.

Handler создаёт сервис один раз на время жизни приложения. У ручной команды
новый процесс и новая map при каждом `go run`. Компоненты повторно используются
между пулами внутри одного запуска, но между запусками ручной команды кэш не сохраняется.

Полный запрос StakeDAO использует до **8 обработчиков токенов**. Ручная команда
1–5 пулов оставляет один обработчик и прежний лимит 5 RPC/с / 300 попыток.
`Snapshot.Workers=0/1` сохраняет последовательный режим; максимальное значение — 8.
Внутри одного `FetchAt` одинаковые `(target, calldata)` на фиксированном блоке
читаются один раз, включая одновременные обращения и неуспешные пробы общего
компонента. Каждый обработчик имеет свой стек зависимостей, поэтому циклические
ERC4626-ссылки не приводят к ожиданию результатов друг друга.

Этот кэш контрактных вызовов существует только до завершения `FetchAt`. Ошибки
не сохраняются в `prices.Store` и проверяются заново при следующем обращении.
Успешные USD-котировки и компоненты сохраняются в Store с прежним TTL/ValidUntil.
Чтения цен остаются **прямыми `eth_call`**, включая оракулы: их перенос через
Multicall мог бы изменить проверки `msg.sender`/`tx.origin`. Все обработчики
используют тот же клиент с общим `RPC_MAX_RPS`; параллелизм не добавляет квоту CU.

HTTP/network-ошибка останавливает обработчиков; `ErrPriceRPCUnavailable` передаёт
её StakeDAO как ошибку RPC. Revert отдельной пробы/отсутствующий USD feed остаётся
ошибкой цены конкретного токена: другие пригодные котировки сохраняются. В логах
`rpc_reverts` выделяет из `rpc_errors` ответы JSON-RPC с кодом 3; это не число
подтверждённых цен. Поля ответа TVL/APR остаются `null`, если цены недоступны.

## Контрактные источники (Ethereum Mainnet)

- Chainlink Feed Registry: `0x47Fb2585D2C56Fe188D0E6ec628a38b74fCeeeDf`.
  `getFeed(token, address(840))` выбирает USD feed; цена читается из
  `decimals()` и `latestRoundData()` самого feed.
- ERC4626: `asset()`, `decimals()` и `convertToAssets(10^shareDecimals)`.
  Цена доли = нормализованное число underlying assets × их USD-цена.
  Поддержаны цепочки ERC4626 до трёх уровней, циклы/нулевые asset отклоняются.
- Plain Curve StableSwap NG: сначала подтверждается регистрация через
  `get_implementation_address(pool)` в канонической Ethereum NG Factory
  `0x6A8cbed756804B16E05E741eDaBd5cB544AE21bf`. Затем читаются
  `is_meta`, `N_COINS`, `coins(i)`, `balances(i)`, `totalSupply`, `decimals`.
  Цена LP = сумма USD-стоимости всех компонентов / нормализованный LP supply.
  `balances(i)` исключает начисленные admin fees, согласно интерфейсу Curve.

Все новые RPC-чтения выполняются через тот же клиент и на том же snapshot-блоке,
что и данные StakeDAO. В ограниченной команде они входят в лимиты 5 RPS и
300 RPC-попыток. `recursivePools`, metapools и вложенные LP здесь не используются;
они требуют отдельной доработки. `virtual_price` не принимается за USD-цену.

## Свежесть и отсутствие данных

Кэш обновляется при следующем обращении после 15 минут, без фонового polling.
`UpdatedAt` контрактной котировки — время snapshot-блока чтения.
Для Chainlink проверяются положительный answer, roundId/answeredInRound,
startedAt/updatedAt, отсутствие будущего timestamp и возраст раунда меньше
25 часов. 25 часов — явная политика приложения (24 часа плюс час запаса),
не автоматически обнаруженный heartbeat произвольного feed.

`ValidUntil` хранит срок годности оракула/зависимостей: чтение кэша не продлевает
его. Производная LP/ERC4626-цена не живёт дольше, чем использованные компоненты.
При отсутствии decimals, USD anchor или корректного supply цена остаётся
недоступной. Не используются предположение `$1`, цена одного компонента вместо
LP, Curve EMA или автоматический HTTP fallback.

При ручной проверке Feed Registry подтвердил USD feeds для USDC, USDT, CRV, CVX,
crvUSD и frxUSD. Для avUSD (`0xf4c13D631450De6B12a19829E37c8e2826891dC4`)
USD feed не найден. Поэтому avUSD/evaUSDC пока сохраняет `tvl/apr: null`.
Оценка через Curve EMA не включена без отдельного решения о таком источнике.

## Проверенные интерфейсы

ABI содержит только используемые view-методы из официальных источников:

- [Chainlink FeedRegistryInterface](https://github.com/smartcontractkit/feed-registry/blob/master/contracts/interfaces/FeedRegistryInterface.sol).
- [AggregatorV3Interface](https://github.com/smartcontractkit/chainlink-evm/blob/develop/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol);
  совместимый ABI также уже находится в `contracts/curve/ChainlinkCRVUSD.json` проекта.
- [OpenZeppelin IERC4626](https://github.com/OpenZeppelin/openzeppelin-contracts/blob/master/contracts/interfaces/IERC4626.sol).
- [CurveStableSwapNG](https://github.com/curvefi/stableswap-ng/blob/main/contracts/main/CurveStableSwapNG.vy).
- [CurveStableSwapFactoryNG](https://github.com/curvefi/stableswap-ng/blob/main/contracts/main/CurveStableSwapFactoryNG.vy).
- [Curve network constants](https://github.com/curvefi/curve-js/blob/master/src/constants/network_constants.ts),
  `ALIASES_ETHEREUM.stable_ng_factory`.

Тесты используют локальный RPC-стенд: нормализация decimals, стоимость ERC4626/LP,
cache hit/miss/expiry, невалидные раунды, отсутствие USD anchor, нулевой supply,
неподдержанные metapool/nested LP и отмена контекста. Они не тратят квоту RPC.
