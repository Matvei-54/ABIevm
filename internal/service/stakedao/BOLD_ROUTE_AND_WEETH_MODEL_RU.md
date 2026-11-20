# BOLD/USDC: маршрут наград; weETH/WETH: модель комиссий

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
