# Карта источников доходности и признаков активности

## Что реализовано

Разделение ответа реализовано в `yieldBreakdown`:

| Компонент | Реализация | Входит в верхний apr |
| --- | --- | --- |
| crv | Чистый поток CRV Locker + подтверждённая доля Sidecar | Да |
| vaultExtra | Текущие распределения RewardVault | Да |
| gaugeExtra | Gross-проекция дополнительных токенов Curve gauge | Нет |
| merkl | Выделены поля и причина недоступности; расчёта пока нет | Нет |
| trading | Окна 7/30 дней для проверенных fixed-rate LP; архив RPC обязателен | Не входит в APR; основной APY за 30 дней |

То есть разделение сделано, но получение всех компонентов ещё не завершено.
`aprScope=vault_rewards`, `aprStatus` и краткий `status` относятся к учтённым
потокам vault. `yieldBreakdown.status=partial` относится к полному составу.
Переключение между API StakeDAO и RPC не реализовано.

## Сначала определить роли контрактов

Для каждого vault адреса проверяются на одном блоке B. Адрес vault в кратком
JSON не является ни Curve gauge, ни LP-токеном, ни адресом оракула.

| Обозначение | Как получить адрес | Для чего |
| --- | --- | --- |
| V: RewardVault | Controller.gauge(G).vault, сверить с запрошенным vault | Активы вкладчиков и actual extra distributions |
| G: Curve gauge | V.gauge(), проверить регистрацию Controller.gauge(G) | Основная эмиссия и дополнительные стимулы |
| A: LP-токен | V.asset(), сверить с Controller.gauge(G).asset | Количество активов и USD-цена LP |
| S: Strategy | V.strategy() | Учёт размещения и адрес Locker |
| L: Locker | S.LOCKER() | working balance основной доли |
| Q: Accountant | V.ACCOUNTANT(), сверить с S.ACCOUNTANT() | Комиссии и учёт supply vault |
| Allocator | V.allocator() | getAllocationTargets(G) — Locker и Sidecar |
| T: Sidecar | Каждый подтверждённый target, отличный от L | Размещение через Convex |
| R: Convex reward pool | T.baseRewardPool() | Эмиссия Convex и его общий staked supply |
| Receiver | Controller.gauge(G).rewardReceiver | Куда поступают дополнительные токены |

Текущие адреса общей Ethereum-интеграции:

- ProtocolController: `0x2d8BcE1FaE00a959354acd9ebf9174337A64d4fb`.
- Curve GaugeController: `0x2F50D538606Fa9EDD2B11E2446BEb18C9D5846bB`.
- Проверенная CurveStrategy: `0xcE37598c3b26eCbdF13356f9E1FFf2Cd36EE602a`.
- Проверенный Accountant: `0x93b4B9bd266fFA8AF68e39EDFa8cFe2A62011Ce0`.
- Convex Booster: `0xF403C135812408BFbE8713b5A23a04b3D48AAE31`.

Это проверенные адреса ролей, а не разрешение считать любой контракт совместимым.
Связи сверяются с регистрацией и фактическим snapshot-состоянием.

## CRV APR: несколько функций, а не getter готового APR

| Контракт | Функции | Какие исходные данные читаем |
| --- | --- | --- |
| G | is_killed() | Можно ли учитывать основную CRV-эмиссию gauge |
| G | inflation_rate(), working_balances(L), working_supply() | Эмиссия и boost через рабочую долю Locker |
| Curve GaugeController | gauge_relative_weight(G, timestampB) | Вес gauge в основной эмиссии |
| T | balanceOf(), baseRewardPool(), pid(), asset(), ACCOUNTANT(), REWARD_TOKEN(), rewardReceiver(), PROTOCOL_CONTROLLER(), BOOSTER() | Собственная доля и проверка связей Sidecar |
| Convex Booster | poolInfo(pid) | Проверить LP, gauge и reward pool |
| R | rewardToken(), rewardRate(), totalSupply(), periodFinish() | Текущая CRV-эмиссия Convex; истёкший период даёт нулевой поток |
| Q | getProtocolFeePercent(), getHarvestFeePercent(), totalSupply(V) | Доли комиссий и учётный supply |
| V | POLICY(), totalAssets() | Политика harvest и количество активов вкладчиков |
| S | balanceOf(G) | Физическое покрытие учётных активов |
| CRV / USD source | decimals(), latestRoundData() | Масштаб токена и пригодная USD-цена |

После нормализации CRV в tokens/sec:

```text
Lrate = inflation × weight × workingBalance(L) / workingSupply
Crate = rewardRate(R) × balanceOf(T) / totalSupply(R)
net = Lrate + Crate − Lrate × protocolFee − (Lrate + Crate) × harvestFee
CRV APR = net × CRV_USD × 31536000 × 100 / vaultTVL_USD
```

Вес нормализуется на 1e18. Raw CRV amounts нормализуются по CRV decimals.
Политика POLICY учитывается как в проверенном `curvePrimaryRate`; для
поддержанного HARVEST-варианта отдельное удержание harvest fee здесь обнуляется.
При нулевом собственном Sidecar balance нельзя использовать общий rewardRate
Convex как эмиссию этого vault. Pending/earned — накопленная сумма, не APR.

Реализация: [curvePrimaryRate / valueRewards](StkDaoSRV.go).

## Дополнительные награды

| Компонент | Контракт | Функции | Расчёт |
| --- | --- | --- | --- |
| vaultExtra | V | getRewardTokens(), rewardData(token) | Взять rewardRate и periodFinish; нормализовать rate на 10^(18+tokenDecimals), умножить на цену и год, разделить на TVL vault |
| gaugeExtra | G | lp_token(), reward_count(), reward_tokens(i), reward_data(token), totalSupply() | Нормализовать global rate только на 10^tokenDecimals; разделить USD-эмиссию на USD-стоимость supply всего gauge |

На токене награды читается `decimals()`. `rewardData` RewardVault и
`reward_data` Curve gauge — **разные ABI с разными масштабами rate**.
У gauge проверяется шестипольный ответ
`(token, distributor, period_finish, rate, last_update, integral)`.

```text
gaugeExtra APR = globalRate × tokenUSD × 31536000 × 100
                / (gaugeSupplyInLP × lpUSD)
```

Это upstream gross-проекция, а не гарантированная выплата держателю vault.
Её нельзя автоматически сложить с vaultExtra, который может быть последующим
этапом выплаты того же токена. Место фактического получения дохода и источник
стимула — разные признаки.

Реализация: [loadRewardStreams](StkDaoSRV.go),
[loadYieldBreakdown / valueYieldBreakdown](yieldBreakdown.go).

## Merkl: что необходимо добавить

Сейчас компонент возвращает `incentive_registry_required` и null. Следующие
интерфейсы — подтверждённые источники для доработки, **не уже включённые чтения**:

| Роль | Функции | Что определяют |
| --- | --- | --- |
| Receiver | rewardRouter(), rewardVault(), legacyRewardReceiver() | Связи маршрута наград; функции зависят от фактической версии |
| Router | merkleDistributor(); входные данные прошлых execute-транзакций и события | Настроенный получатель и выбранный путь средств; execute не вызывается для чтения |
| StakeDAO incentive distributor `0xd4898a378ea555595c4e7dbde722b134a3f346d1` | nbIncentives(), incentives(id) | Gauge, token, amount, start/end, duration и источник кампании |
| Тот же distributor | root(), pendingRoot(), lastSetRoot(), claimed(user,token) | Состояние опубликованного распределения, не getter APR |

Верифицированный [ABI distributor](https://etherscan.io/address/0xd4898a378ea555595c4e7dbde722b134a3f346d1#code)
проверен при составлении карты. В его incentives нужен индекс, а не адрес токена.
Для актуальной кампании ожидаемая ставка получается из amount/duration и цены,
но перед агрегацией нужно подтвердить получателей, пригодность кампании и
исключить дубликаты gauge/vault/Onlyboost. История владения нужна для точной
выплаты пользователю; текущая оценка APR не заменяет расчёт claimable.

Важно: поле Router с именем merkleDistributor не гарантирует, что адрес
поддерживает ABI distributor. У PYUSD/crvUSD на проверенном ранее блоке оно
возвращало AllMight `0xDBd24b092f686b12650EC1450e3A7138F714506c` — исполнитель,
не тот же контракт, что incentive distributor выше. Нужна трассировка перевода,
а не подстановка одного адреса вместо другого.

## Trading Fees APY: готового универсального getter нет

Функция fee() возвращает комиссию сделки, не годовую доходность LP.
Один snapshot balances/totalSupply также не показывает заработанные за неделю
комиссии. Поэтому нельзя указать одну функцию x, которая уже даёт наш Trading APY.

Для onchain-реализации нужна история, соответствующая фактической версии
Curve-пула, и согласованное окно. Добавленная ограниченная модель читает
get_virtual_price(), initial_A(), future_A(), initial_A_time(), future_A_time()
и coins(i) на двух блоках; для NG также проверяет типы активов и stored_rates().
Рост virtual price или ERC4626 conversion может включать underlying yield,
а не только trading fees, поэтому вне проверенных fixed-rate моделей этой
подписью не используется.

Исторический публичный reader StakeDAO использовал `latestWeeklyApy` из Curve
subgraph; V2 reader разбирает `TRADING_FEES` из аналитических данных gauge.
Это внешняя аналитика, не универсальный метод контракта. В нашем приложении
нет её HTTP-поставщика. Вместо импорта общего base APY реализована ограниченная
onchain-модель fixed-rate fee growth: [TRADING_HISTORY_RU.md](TRADING_HISTORY_RU.md).
Прочие модели по-прежнему оставляют APY неизвестной с причиной.

## Откуда USD-цены для любого APR

| Случай | Контракт / функции |
| --- | --- |
| Chainlink USD | FeedRegistry `0x47Fb2585D2C56Fe188D0E6ec628a38b74fCeeeDf`: getFeed(token,USD); feed: decimals(), latestRoundData() |
| Старый Curve LP | MetaRegistry `0xF98B45FA17DE75FB1aD0e7aFD971b0ca00e379fC`: get_pool_from_lp_token, get_lp_token, is_meta, get_n_coins, get_coins, get_balances, get_decimals; LP: totalSupply, decimals |
| Curve NG LP | Проверенный factory/pool: N_COINS, coins, balances, totalSupply, decimals; отдельные USD-цены компонентов |
| ERC4626 share | asset(), decimals(), convertToAssets(10^decimals); затем USD-цена underlying |
| Закреплённые токены | [pinnedRoutes.go](../prices/pinnedRoutes.go): weETH cross Chainlink, frxETH/BOLD Curve EMA и USD-anchor |

Это цена LP по NAV, а не доходность комиссий. Формула LP_USD = сумма стоимости
активов pool / выпущенные LP. Неподдержанные маршруты, stale-ответ или отсутствие
цены компонента оставляют USD-оценку недоступной. Цена не подставляется как 1 USD.

## Null и активность — разные проверки

| Признак | Чтение | Что означает и как использовать |
| --- | --- | --- |
| Регистрация / shutdown | Controller.gauge(G), isShutdown(G) | Сверить vault/asset/protocol; shutdown уже исключается текущим приложением |
| Пауза протокола | Controller.isPaused(protocolId) | Отдельный текущий флаг протокола; не возраст пула и не доказательство отсутствия всех доходов |
| Основная CRV-эмиссия отключена | G.is_killed() | Учитывается в CRV; не доказывает отсутствия торговых комиссий/extra incentives |
| Нет веса CRV | GaugeController.gauge_relative_weight(G,tB)=0 | Нулевой текущий CRV-поток; пул не обязательно закрыт |
| Пустой vault | V.totalAssets()=0 | Нет активов вкладчиков этого vault, APR с его TVL как знаменателем не определён |
| Пустой gauge | G.totalSupply()=0 | Другая метрика: все позиции gauge, не только StakeDAO vault |
| Истёк период конкретного токена | periodFinish<=timestampB | Окончилась эта награда; не весь пул |
| Недоступен расчёт | aprReasons / component.reasons | Ошибка/пробел данных или неподдержанная модель, не сигнал закрытия |

`maxDeposit` нельзя без проверки использовать как универсальный индикатор
открытого vault: у фактического RewardVault этих двух примеров функция pure
и всегда возвращает type(uint128).max. Она не проверяет shutdown или pause.

Старый по дате создания пул может оставаться ликвидным. Фильтр по возрасту,
APR=null или APY=null скроет такие пулы. Для списка разумно отдельно учитывать
подтверждённый shutdown, пустоту, минимальный TVL и полноту расчёта. Пауза,
is_killed и нулевой вес должны иметь собственные признаки/политику интерфейса,
а не превращаться в единый статус «неактивен».
Отдельные nullable-признаки теперь реализованы в `state`, без единого флага
active и без новых фильтров: [POOL_STATE_RU.md](POOL_STATE_RU.md).

## Проверка двух vault из вопроса

Ethereum block **26442996**, timestamp **1795017527**, тот же hash в конце обоих
прогонов. 4 RPC проверки активности + 46 RPC diagnostic endpoint через PublicNode,
без GetBlock CU. Это новый срез, не доказательство состояния в момент старого JSON.

| Vault | Учтённые LP | Shutdown | Gauge killed | CRV weight | Причина null APR |
| --- | ---: | --- | --- | ---: | --- |
| sdYND/YND `0x147367ed8966a46769f331a1210c952babe87b09` | 5751450.021721042504862817 | false | false | 0 | asset_price_missing |
| USD3/sDAI `0x8f082eeb435efc24ccf6a84aced3770b90960728` | 0 | false | false | 0 | empty_vault |

Controller.isPaused(CURVE)=false. У обоих регистрация соответствует запрошенному
vault. sdYND/YND не пустой: не найдена пригодная USD-цена LP, поэтому TVL и APR
остались null даже при прочитанном нулевом CRV rate. USD3/sDAI пуст только как
StakeDAO vault; в самом gauge осталось 1.101544425540554027 LP. Нельзя объявлять
Curve-пул ликвидированным на основании пустого StakeDAO vault.

Gauge-адреса для ограниченного повторения:

- sdYND/YND: `0x2BA46635Be14Ea6d3b27f0ed3C3881dC95AF8a13`.
- USD3/sDAI: `0x2CBEecEE6140e6392Ac770D0F30D80d7EFD82240`.

```text
GET /reservesStakedao/ethereum/diagnostics?gauges=0x2BA46635Be14Ea6d3b27f0ed3C3881dC95AF8a13,0x2CBEecEE6140e6392Ac770D0F30D80d7EFD82240&strictPrices=true
```

Результаты: [activity.json](../../../examples/stakedao-source-map/activity.json)
и [2vaults.json](../../../examples/stakedao-source-map/2vaults.json).

## Следующие шаги

1. Признаки жизненного состояния выделены в state; отдельно можно выбрать
   политику отображения empty/shutdown/paused/killed без фильтра по null.
2. Индексировать и проверять активные Merkl-кампании и маршруты выплат,
   с курсором и правилами удаления дублей.
3. Подтвердить положительные исторические APY на архивном RPC и расширять
   поддержку trading fees за пределы четырёх проверенных fixed-rate пулов.
4. Расширять проверенные ценовые маршруты отдельно от фильтрации пулов.

Карта включает последующую реализацию fee APY с окнами 7/30 дней. Её проверки
и ограничения описаны в [TRADING_HISTORY_RU.md](TRADING_HISTORY_RU.md).
БД и фильтры не менялись; полная загрузка mainnet реестра не выполнялась.
