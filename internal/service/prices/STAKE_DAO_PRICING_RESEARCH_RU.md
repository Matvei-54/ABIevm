# Источники цен StakeDAO: проверка документации и кода

Проверено 29 октября 2026 года. Это исследование и план; источники цен приложения
не переключены. Запросов к пользовательскому RPC и расхода его CU не было.

## Что удалось подтвердить

StakeDAO использует несколько путей получения цен. Опубликованный SDK сочетает
контрактные балансы и supply с внешними USD-источниками. Отдельно существуют
контрактные LP-оракулы для lending. Универсального подтверждённого getter,
возвращающего USD-цену любого из наших LP, в рассмотренных источниках не найдено.

### SDK и API

Изучены опубликованные пакеты `@stake-dao/reader@0.5.52` и
`@stake-dao/constants@0.5.21`. API указывает reader `^0.5.52`, а reader зависит от
constants `0.5.21`; на момент проверки это последние опубликованные версии.
Архивы скачаны из npm для чтения; checksum сверены с registry metadata, код не
запускался и пакеты в приложение не устанавливались.

В `reader/src/strategies/curve/fetch/curveApiData/getCurveApiData.ts`:

1. Через контрактные registry читаются LP, состав, decimals, балансы и totalSupply.
2. Цены компонентов загружаются из DeFiLlama; есть дополнение из Curve price API
   и отдельные случаи GeckoTerminal.
3. Рассчитывается `usdTotal = Σ(normalizedBalance × componentUsdPrice)`.
4. `lpPriceInUsd = usdTotal / normalizedTotalSupply`.

В `reader/src/prices.ts` общий путь `getPrices` начинается с заранее настроенных
`extensions.onChainPrice`, затем использует CoinGecko, DeFiLlama и GeckoTerminal.
`getPricesFromCurve` обращается к `https://prices.curve.finance/v1/usd_price/{chain}`.
Набор источников различается между этой общей функцией и Curve-specific путём.

В `api/scripts/updatePrices.ts` LP-цены из стратегий добавляются в общую таблицу
цен. Текущий публичный `api/src/lib/strategies/v2.ts` получает данные V2 из Hub.
Это подтверждает использование backend-слоя; публичный SDK не доказывает точную
внутреннюю реализацию сегодняшнего Hub. Сам Hub из данной среды недоступен (403).

В SDK есть поведение, которое переносить в наш строгий расчёт не следует:
отсутствующая цена компонента превращается в нулевой `usdBalance`, после чего
сумма остальных компонентов может дать положительный LP price. В приложении
неполный состав не должен представляться полной достоверной USD-оценкой.

### Сопоставление с пользовательским JSON

Снимок пользователя: Ethereum, блок **26298688**, 359 пулов:

- 155 с нулевым количеством активов;
- 204 с положительным количеством;
- у 159 непустых отсутствует TVL;
- у 45 непустых есть числовой TVL/APR: 30 `available`, 15 `partial`.

В публичном файле `api/strategies/v2/curve/1.json` найдено 155 совпадений по
`lpToken.address` для этих 159 проблемных asset; у 154 есть положительный
`lpPriceInUsd`. Это проверка покрытия по адресам, а не точности цены/доходности:
снимки не привязаны к одному блоку, данные API не верифицировались контрактами.

## Найденные адреса Ethereum Mainnet

| Назначение | Адрес | Подтверждение и применимость |
| --- | --- | --- |
| Curve MetaRegistry | `0xF98B45FA17DE75FB1aD0e7aFD971b0ca00e379fC` | Официальный address book; ABI Etherscan содержит `get_pool_from_lp_token`, `get_coins`, `get_balances`, `get_decimals`, `get_lp_token`. Это registry состава, а не готовая USD-цена. |
| FXS USD feed | `0x6Ebc52C8C1089be9eB3945C4350B68B8E4C2233f` | `extensions.onChainPrice` токена FXS в constants; кандидат для явной конфигурации oracle. |
| USDS USD feed | `0xfF30586cD0F29eD462364C7e81375FC0C71219b1` | Та же конфигурация constants для USDS. |
| USDf USD feed | `0xb177857a1799aA5F7fEb5799Fdf12CbE8fdF78B1` | Та же конфигурация constants для USDf. |
| Legacy oracle рынка USDC/crvUSD → USDC | `0x3bb43EF3EBF21E84c38e958fbF85af482d594C77` | Address book + опубликованный ABI/constructor Etherscan. `BASE_ASSET` — LP `0x4DEcE678ceceb27446b35C672dC7d61F30bAD69E`. |
| Oracle рынка frxUSD/sDOLA → frxUSD | `0x0c63A83a404AE7FA679A35F98C38C05A60Aa820C` | Address book; ABI Etherscan содержит `BASE_ASSET`, `QUOTE_ASSET`, `price`, `decimals`, параметры feeds. Конфигурацию нужно читать перед использованием. |

Адреса feeds из constants не являются подтверждением их свежести на нужном блоке.
SDK использует для них `latestAnswer()` с заданными decimals; в приложении нужны
проверки применимого интерфейса, единиц и свежести, включая `latestRoundData` там,
где он поддержан. В этой проверке живые цены контрактов не запрашивались.

### Конкретный кандидат из JSON

Пул `Stake DAO Curve.fi Factory Plain Pool: crvUSD/USDC Vault`:

- gauge: `0x95f00391cb5eebcd190eb58728b4ce23dbfa6ac1`;
- asset: `0x4dece678ceceb27446b35c672dc7d61f30bad69e`;
- assetAmount: `3766505.165368452189945430`, `tvl/apr: null`.

Constructor oracle `0x3bb4…4C77` прямо задаёт этот pool, quote asset USDC,
нулевой quote feed, пустую цепочку USD feeds и scaling exponent 36.
Значит результат здесь выражен в **USDC**, а не автоматически в USD.
Для этого варианта `decimals = 36 + 6 − 18 = 24`; после нормализации нужна
USD-цена самого USDC. В общем случае следует читать `decimals()` и конфигурацию,
а не фиксировать это число для всех oracle.

## Как устроены lending-oracles

[Документация Oracles](https://docs.stakedao.org/lending/oracles) описывает
консервативную StableSwap-оценку:

```text
LP в coin0 = min(1, EMA цен остальных coins относительно coin0) × virtual_price
LP в quote = LP в coin0 × цепочка coin0→USD / USD-цена quote
```

Это не та же оценка, что сумма USD-балансов / LP supply: min-clamp предназначен
для стоимости залога. Если использовать её как TVL, рассчитанный APR также
изменится. Метод оценки нужно выбрать явно.

Публичный audit-extraction содержит `CurveStableswapOracle`,
`CurveCryptoswapOracle`, `BaseOracle` и adapters:

- StableSwap: `price_oracle` и `get_virtual_price`;
- CryptoSwap: `lp_price()` с дальнейшей конвертацией валюты;
- `BaseOracle.price()`: цена LP в `QUOTE_ASSET`; `decimals()` определяет масштаб;
- `OracleChainlinkAdapter`: предоставляет `latestRoundData/decimals`, но его
  timestamp синтетический — свежесть исходных feeds проверяется нижележащим oracle.

Документация пока ограничивает основной описанный путь StableSwap-NG и отдельно
оговаривает версии. Наличие CryptoSwap-класса в исходниках не подтверждает
развёрнутый oracle для каждого crypto LP. Ссылка документации на contracts-monorepo
возвращает 404; использовано опубликованное извлечение для аудита с provenance.
Оно не является доказательством соответствия каждого deployed address этой версии.

## Предлагаемый порядок доработки

1. Сохранить один `TokenPriceService` и текущий кэш. Добавить проверенную явную
   конфигурацию `chain/token → USD feed` как дополнение к Feed Registry: наличие
   адреса в SDK позволяет избежать части проб, но требует проверки свежести.
2. Расширить определение Curve pool через MetaRegistry: LP и pool могут иметь
   разные адреса. Добавлять адаптеры семейств постепенно, с нормализацией supply
   и балансов. Metapools/`recursivePools` остаются отдельной задачей.
3. Первым проверить адреса/единицы oracle для `crvUSD/USDC` и сравнить его оценку
   с полной балансовой оценкой на одном блоке. Определить, подходит ли выбранная
   семантика для TVL/Reward APR; EMA-путь пока автоматически не включать.
4. Для контроля использовать API как внешний ориентир. Переключение приложения
   на HTTP-цены — отдельное решение: сейчас пользователь выбрал blockchain-источники.
5. Затем проверить 3–5 разных семейств пулов ограниченными вызовами, без очередного
   полного сканирования и без предположения, что неизвестная цена равна нулю/$1.

## Источники и зафиксированные версии

- [Strategies APR](https://docs.stakedao.org/st-apr): Current и Projected APR,
  примеры протокольных API. Это не обещание совпадения с нашим `aprScope=rewards`.
- [Oracles](https://docs.stakedao.org/lending/oracles).
- [Addresses](https://docs.stakedao.org/addresses).
- [Официальный address book](https://github.com/stake-dao/offchain-registry/blob/4213ad0455a78cfa36c326ae3c48f0bd37568b05/data/address-book/README.md).
- [SDK reader 0.5.52](https://www.npmjs.com/package/@stake-dao/reader/v/0.5.52),
  опубликован 2026-09-15; constants 0.5.21 — 2026-08-31. Пути исходников приведены выше.
- [API: обновление цен](https://github.com/stake-dao/api/blob/60745b7b18cf138b1f2122f88eee6204abd85f53/scripts/updatePrices.ts).
- [API: источник V2](https://github.com/stake-dao/api/blob/60745b7b18cf138b1f2122f88eee6204abd85f53/src/lib/strategies/v2.ts).
- [API: снимок Ethereum V2](https://github.com/stake-dao/api/blob/60745b7b18cf138b1f2122f88eee6204abd85f53/api/strategies/v2/curve/1.json),
  commit 2026-10-29 15:22:43 UTC; дата commit не заменяет время исходных цен.
- [Audit-extraction: BaseOracle](https://github.com/stake-dao/lending-contracts-audit/blob/da08e868037036871bc36843b0e291c5959422ca/src/lending/oracles/BaseOracle.sol),
  [StableSwap](https://github.com/stake-dao/lending-contracts-audit/blob/da08e868037036871bc36843b0e291c5959422ca/src/lending/oracles/CurveStableswapOracle.sol),
  [CryptoSwap](https://github.com/stake-dao/lending-contracts-audit/blob/da08e868037036871bc36843b0e291c5959422ca/src/lending/oracles/CurveCryptoswapOracle.sol).
- [Etherscan: USDC/crvUSD oracle](https://etherscan.io/address/0x3bb43EF3EBF21E84c38e958fbF85af482d594C77#code).
- [Etherscan: frxUSD/sDOLA oracle](https://etherscan.io/address/0x0c63A83a404AE7FA679A35F98C38C05A60Aa820C#code).
- [Etherscan: Curve MetaRegistry](https://etherscan.io/address/0xF98B45FA17DE75FB1aD0e7aFD971b0ca00e379fC#code).
