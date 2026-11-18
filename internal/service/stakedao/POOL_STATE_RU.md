# Состояние пула отдельно от APR

В подробном пуле появился `state`. Он содержит только факты на snapshot-блоке,
не зависит от USD-цен и не вычисляет единый флаг active по косвенным признакам.
`aprStatus`, `aprReasons`, `yieldBreakdown` и краткий `status` сохраняют прежний
смысл — полнота расчёта соответствующей доходности.

| Поле state | Источник | Значение |
| --- | --- | --- |
| blockNumber | Snapshot запроса | Все признаки относятся к этому блоку |
| registered | ProtocolController.gauge(gauge) | Есть регистрация vault; в обычном списке она уже сверена с vault/asset/receiver/protocol |
| shutdown | Последний bool в ProtocolController.gauge(gauge) | Отключение gauge в StakeDAO |
| protocolPaused | ProtocolController.isPaused(protocolId) | Текущая пауза протокола в StakeDAO |
| gaugeKilled | Curve gauge.is_killed() | Отключение основной CRV-эмиссии, не всех источников дохода |
| hasAssets | RewardVault.totalAssets() | Raw количество активов вкладчиков положительно; USD-цена и decimals не нужны |
| crvRelativeWeight | Curve GaugeController.gauge_relative_weight(gauge,timestampB) | Точный вес CRV как decimal string от 0 до 1, после деления raw на 1e18 |
| hasCrvWeight | Положительность прочитанного crvRelativeWeight | Нулевой вес не означает закрытие пула |
| issues | Результаты отдельных getter | field, code, contract, method; без URL RPC и credentials |

Флаги имеют три состояния: true, false, null. Непрочитанный getter не становится
false. У незарегистрированного gauge остальные поля неприменимы и остаются null.
Для протокола с неподдержанным Curve ABI поля gaugeKilled/CRV weight остаются
null с причиной unsupported_protocol. Отсутствие количества активов даёт
hasAssets=null, а не «vault пустой». При непрочитанных decimals нормализованное
assetAmount может быть null, но hasAssets остаётся известным из raw totalAssets.

У killed gauge могут остаться LP, торговые комиссии и другие incentives.
Pause не означает пустоту; ненулевой вес может сохраняться даже при killed=true.
Завершение периода конкретной награды тоже не закрывает весь пул.
Новых автоматических фильтров по этим флагам, возрасту или null APR не добавлено.

## Где увидеть

- `/reservesStakedao/ethereum?details=true`: state рядом с APR и TVL каждого
  возвращённого пула. Запрос загружает весь реестр.
- `/reservesStakedao/ethereum/diagnostics?gauges=...`: ограниченные 1–5 пулов,
  как раньше, теперь со state. Можно задать block/strictPrices.
- Новый `/reservesStakedao/ethereum/state?gauges=...`: только признаки состояния,
  **без цен, APR, discovery и Postgres**. Можно задать положительный block.
  Возвращает и shutdown-пулы, и незарегистрированные gauge, которые обычная
  загрузка не включает. Для незарегистрированного адреса это запись с
  registered=false, а не выдуманный vault с нулевыми активами.

Последний endpoint сохраняет ограничение 1–5 уникальных ненулевых gauge,
5 RPC/с и 300 попыток. Чтения объединены Multicall, повторные getters одного
snapshot переиспользуются. В конце проверяется hash блока. Транспортный отказ
или реорганизация не выдаются за подтверждённое состояние.

Пример для двух vault из вопроса:

```text
GET /reservesStakedao/ethereum/state?gauges=0x2BA46635Be14Ea6d3b27f0ed3C3881dC95AF8a13,0x2CBEecEE6140e6392Ac770D0F30D80d7EFD82240
```

Нужны gauge-адреса, не поле address краткого ответа (оно содержит vault).
При заданном block все признаки проверяются на нём. Отсутствующий block
выбирает текущую вершину. Это onchain-наблюдения, не гарантия успешности
будущего deposit/withdraw: нет единственного универсального getter для всех
ограничений и версий контрактов. В частности maxDeposit этих RewardVault
возвращает константу и не подходит для определения активности.

## Реальная проверка

Блок **26443121**. Новый HTTP handler проверен через PublicNode на двух gauge
и адресе `0x0000000000000000000000000000000000000001` без регистрации: **9 RPC**.
Подробная диагностика двух зарегистрированных пулов выполнена на том же блоке:
**46 RPC**, результат state совпал, snapshot hash подтверждён.
GetBlock CU не использовались; полный реестр не запускался.

| Vault | registered | shutdown | paused | killed | hasAssets | hasCrvWeight | Причина отсутствия APR |
| --- | --- | --- | --- | --- | --- | --- | --- |
| sdYND/YND | true | false | false | false | true | false | asset_price_missing |
| USD3/sDAI | true | false | false | false | false | false | empty_vault |

У незарегистрированного gauge registered=false, остальные флаги null.
Результаты: [state.json](../../../examples/stakedao-pool-state/state.json),
[diagnostics.json](../../../examples/stakedao-pool-state/diagnostics.json).

## Для чего нужны реестр Merkl и история комиссий

Реестр Merkl — проверенный список кампаний: ID, gauge/vault, токен, сумма,
период start/end, источник и контракт распределения. Он нужен, чтобы узнать,
какая дополнительная награда относится именно к этому vault **сейчас**, и не
прибавить истёкшую, чужую или уже учтённую через gauge/vault кампанию.
Это не полная копия блокчейна и не обязательно новая большая БД.
После проверки условий amount/duration и USD-цена дают оценку годового потока;
точная выплата пользователю дополнительно зависит от правил и истории позиции.
Из одного RewardVault.rewardData все внешние кампании обнаружить нельзя.

История комиссий нужна для строки Trading Fees (APY). Процент комиссии сделки
и доход LP за год — разные величины. Для последнего нужно знать, сколько
комиссий реально заработано за выбранное окно, например неделю, и корректно
учесть капитал и изменения позиции. Текущие balances и оракул цены не содержат
всей этой информации. Рост цены LP может также включать underlying yield,
поэтому не равен автоматически торговой доходности.

Эти два источника позволят заполнить отдельные отсутствующие компоненты и
приблизить состав отчёта к UI StakeDAO. Они не определяют, существует ли пул,
и не устраняют любую причину null APR. В этой правке их сбор и расчёт не
реализовывались; переключение API StakeDAO → RPC также не добавлялось.

## Проверки

Локальными тестами покрыты: состояние при отсутствующей цене и APR, независимые
pause/killed/weight/hasAssets, нулевые и неизвестные значения, malformed getters,
shutdown и незарегистрированные адреса, валидация 1–5 gauge, отсутствие вызовов
цен/discovery в state-only, фиксированный блок и реорганизация, HTTP-маршрут,
шесть полей краткого ответа, отсутствие RPC credentials и Multicall batching.
