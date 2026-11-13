# Состав доходности StakeDAO

Подробный ответ содержит `yieldBreakdown`. Краткий список сохраняет шесть полей.
Верхний `apr` по-прежнему складывает рассчитанные primary CRV и активные потоки
RewardVault; область явно названа `aprScope: "vault_rewards"`.

| Поле | Что означает | Прибавляется к верхнему APR |
| --- | --- | --- |
| `crv` | Чистый поток CRV vault после учитываемых комиссий | Да |
| `vaultExtra` | Сумма текущих распределений RewardVault по rate/periodFinish | Да |
| `gaugeExtra` | Текущая gross-оценка дополнительных наград Curve на один staked LP | Нет |
| `merkl` | Отдельная оценка активных кампаний Merkl | Пока недоступна |
| `trading` | Историческая торговая APY пула | Пока недоступна |

`includedInApr: ["crv", "vaultExtra"]` фиксирует правила агрегации.
`yieldBreakdown.status: "partial"` означает, что общий состав доходности не полон.
Верхний `aprStatus` и краткий `status` относятся только к `vault_rewards`.
Доступность CRV не доказывает доступности остальных компонентов.

## Gauge — отдельная оценка

Подтверждённые getters Curve читаются на том же snapshot-блоке через Multicall:
`lp_token`, `totalSupply`, `reward_count`, `reward_tokens`, `reward_data`.
Проверяются соответствие LP регистрации Controller, не более восьми уникальных
ненулевых токенов, distributor, timestamp, periodFinish, decimals и USD-цены.

ABI `reward_data` подтверждён официальным
[Curve factory gauge](https://github.com/curvefi/curve-factory/blob/5272fd2b1199f185d9cd94822be9262c387dc2c2/contracts/LiquidityGauge.vy):
`(token, distributor, period_finish, rate, last_update, integral)`.
Несовместимые getters оставляют компонент недоступным; другой ABI не угадывается.

```text
gauge APR = globalTokenRatePerSecond × tokenUSD × 31536000 × 100
            / (gaugeTotalSupplyInLP × lpUSD)
```

В отличие от RewardVault, gauge rate не имеет дополнительного масштаба `1e18`:
деление выполняется только на `10^tokenDecimals`. `globalRatePerSecond` в JSON —
эмиссия всего gauge, а `totalSupply` — нормализованное количество стейкнутых LP.
Это не ставка выплаты конкретному vault, не чистая ставка после его комиссий
и не гарантия, что держатель vault получит указанную сумму.

У каждого токена есть `distributionStatus`: active, expired, not_started,
zero_rate или unavailable. Ошибка чтения или цены не заменяется нулём.
Пустой успешно прочитанный список подтверждает нулевой gauge-extra APR.
Компонент может быть `partial`, если оценена только часть токенов.

Одни награды проходят несколько этапов: gauge → receiver → vault. Автоматическое
сложение gaugeExtra и vaultExtra посчитало бы один поток дважды. Gross-проекция
показывается для сравнения со строками сайта `TOKEN APR - Curve gauge`, но не
меняет верхний APR и `rewards` фактических потоков vault.

## Merkl и торговая APY

Merkl возвращает `apr: null`, `status: unavailable`,
`reasons: ["incentive_registry_required"]`. Для реализации нужен подтверждённый
список кампаний с gauge/vault, reward, периодом и суммой, а также правила
пригодности и исключения дубликатов. Чтение только RewardVault не устанавливает
наличие или отсутствие Merkl. Полное сканирование incentives не добавлено.

Trading возвращает `apy: null`, `windowDays: null`, `status: unavailable`,
`reasons: ["historical_window_required"]`. По одному snapshot балансов нельзя
вычислить доходность комиссий за неделю. Нужна отдельная модель истории комиссий
с выбранным окном и отделением доходности underlying и депозитов/выводов.

Ни один из этих null не означает 0%. Общий итог, смешивающий APR и APY, пока
не вычисляется. Источники StakeDAO API/Hub/публикуемых JSON и автоматический
API→RPC fallback в runtime не добавлены.

## Проверка через Swagger

Ограниченный запрос для BOLD/USDC без загрузки истории и Postgres:

```text
GET /reservesStakedao/ethereum/diagnostics?gauges=0x07a01471fA544D9C6531B631E6A96A79a9AD05E9&strictPrices=true
```

Разделение доступно также для стандартной диагностики двух/пяти пулов и
`GET /reservesStakedao/ethereum?details=true` (последний загружает весь реестр).

На блоке **26407069** через PublicNode проверены три пула: **110 RPC,
21.983 секунды**, без GetBlock CU и без ошибок полей:

| Vault | CRV APR | Vault extras APR | Gross gauge extras APR |
| --- | ---: | ---: | ---: |
| weETH/WETH | 1.142647% | 0% | 0% |
| BOLD/USDC | 1.422135% | 0% | 3.504453% |
| frxETH/WETH | 1.130349% | 0% | 0% |

У BOLD дополнительно зарегистрирован LUSD, но его период закончился; APR=0
подтверждён сроком. В BOLD RewardVault распределение не запущено. Положительный
gauge APR и нулевой vault-extra поэтому не противоречат друг другу.
Эта разница не закрывается искусственным прибавлением gauge к vault.

Сохранён [mainnet JSON](../../../examples/stakedao-yield-components.json).
На его данных независимо проверяются LP TVL, gross gauge rate, чистый CRV APR
и отсутствие повторного учёта. Полный mainnet реестр не запускался.

Тесты проверяют 6-decimal rates, общий и отдельный vault supply, один токен в
двух каналах, отсутствие цены/интерфейса/данных, ошибочную пару LP, повторные
токены, ограничение списка, нулевой supply, будущий timestamp, окончание периода,
независимость компонентов, компактную схему HTTP и сравнение Multicall.
