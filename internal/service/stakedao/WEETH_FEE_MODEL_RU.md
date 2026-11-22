# Модель weETH/WETH и диагностика наград BOLD

## Изменения

`weethFeeModel.go` повторяет принцип чтения обменов и нормализации комиссий на
капитал из пользовательского `internal/service/curveService.go`. Исходный файл
Curve не менялся. AAVE также не менялась. Добавлен отдельный код StakeDAO,
не импортирующий глобальные map, клиент или эвристики decimals старого сервиса.

Это первая **оценочная модель fee APR по swaps**, не реконструкция фактической
исторической торговой APY. Оценка явно помечена и не заполняет верхние apr/apy,
не складывается с CRV, gaugeExtra или Merkle. Существующая fixed-rate APY
BOLD/frxETH и её окна 7/30 остаются прежними.

## Запуск в Swagger

Endpoint: `GET /reservesStakedao/ethereum/diagnostics`.

Для weETH/WETH:

```text
gauges = 0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2
weethFeeDays = 1
strictPrices = true
block = оставить пустым для нового среза
```

```text
/reservesStakedao/ethereum/diagnostics?gauges=0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2&weethFeeDays=1&strictPrices=true
```

Допустимы 1, 7 или 30 дней. Без `weethFeeDays` сканирование swaps вообще не
запускается. При включении нужен **только один указанный weETH gauge**:
другой gauge, смесь пулов или нестандартное окно возвращают HTTP 400 до RPC.
`limit` при явном gauges не используется. Нельзя оставлять weethFeeDays при
переключении Swagger на BOLD.

Начинать разумно с суток. Неделя/месяц увеличивают число log-запросов и CU;
они не запускаются автоматически. RPC должен разрешать eth_getLogs по 1500
блоков за выбранный период. Старые contract reads также нужны при явном block.
Ошибка доступа/истории не становится нулевой оценкой. API→RPC fallback нет.

CLI:

```bash
go run -mod=readonly ./cmd/stakedao-check -diagnostics -limit 1 \
  -gauges 0x053df3e4D0CeD9a3Bf0494F97E83CE1f13BdC0E2 \
  -weeth-fee-days 1 -strict-prices -output weeth.json
```

## Формула и единицы

Читаются только проверенные pool/coins: LP
`0xdb74dfdd3bb46be8ce6c33dc9d82777bcfc3ded5`, WETH и weETH, decimals=18.
Factory asset types должны быть [0,1], WETH stored rate=1e18, admin_fee=50%.
Данные coins/balances/stored_rates/dynamic_fee/admin_fee читаются на конечном
snapshot-блоке; shared price cache для этой оценки не используется.

```text
f = pool.dynamic_fee(0,1) / 1e10             # snapshot quote
a = pool.admin_fee() / 1e10                 # 0.5
q_j = stored_rates[j] / 1e18
capitalETH = sum(balance_j / 1e18 * q_j)
estimatedFeeETH = sum(netTokensBought / 1e18 * f/(1-f) * (1-a) * q_bought)
estimatedAPR = estimatedFeeETH / capitalETH * 31536000/elapsedSeconds * 100
```

tokens_bought из TokenExchange уже после комиссии: поэтому f/(1-f), а не
просто f. Берётся только одна сторона обмена, без двойного сложения sold/bought.
Расчёты выполняются decimal с явной точностью деления. ETH здесь — единица
пула с нормализацией oracle rate, не USD и не цена weETH, приравненная к WETH.

**Предпосылки оценки:** одна динамическая fee-quote, rates и капитал конечного
блока применяются ко всему прошлому окну. Реальная NG-комиссия зависит от
состояния и размера каждого обмена; dynamic_fee на конце блока её не
восстанавливает. Изменения fee/rate/капитала за период здесь не replay-ятся.
Ввод/вывод ликвидности и их комиссии в оценку не включены. Рост weETH-rate
сам по себе не прибавляется как торговый доход. Результат не является точным
доходом исторического LP или обещанием годового дохода.

Отдельный результат: `pools[].diagnostics.weethFees`:

- status=estimate_available / unavailable, model и assumptions;
- estimatedApr, estimatedLpFeesEth, referenceCapitalEth;
- dynamicFeeFraction/adminFeeFraction конечного блока;
- windowDays, start/end block+hash, coveredSeconds;
- rangesTotal/rangesDone, swaps, reasons при отказе.

`swaps` теперь nullable: при непрочитанной/непроверенной истории — null,
при полностью проверенной пустой истории — 0. Это различает отсутствие данных
и отсутствие обменов. Прочитанные referenceCapitalEth/dynamicFeeFraction/
adminFeeFraction сохраняются даже при последующем отказе log-истории.

### Диагностика отказа провайдера

В responseWETH_1795189937044.json пользователя, block 26457319, прочитаны
0/5 диапазонов и возвращено swap_history_unavailable. Это не нулевая fee APR.
В старом коде ошибка провайдера заменялась общей причиной, поэтому ни точный
тип отказа, ни тарифное ограничение из того JSON установить нельзя.

Теперь `diagnostics.weethFees.logFailure` сохраняет безопасную классификацию,
метод eth_getLogs, fromBlock/toBlock и доступные числовые httpStatus/rpcCode.
Причины различают rpc_log_range_limit, rpc_log_result_limit, rpc_rate_limited,
rpc_quota_exhausted, rpc_access_denied, archive_rpc_required и timeout/cancel.
Неизвестный отказ остаётся swap_history_unavailable. Текст провайдера,
error data, URL и ключи не выводятся.

Если провайдер **явно сообщил числовой** лимит диапазона, дополнительно
показываются providerMaxBlockRange и requiredRangesForWindow. Лимит не
угадывается по бренду RPC или одному коду -32600. Например, при лимите 10
блоков на окно пользователя из 7177 блоков потребовалось бы 718 log-запросов,
помимо остальных чтений. Это условный пример, а не подтверждённый ответ его
Alchemy endpoint. Автоматического дробления на сотни запросов нет;
ограничения 5 RPS/300 RPC сохраняются.

Для уточнения причины после обновления достаточно повторить тот же суточный
запрос; выбирать 7/30 дней до устранения отказа не нужно. Новый logFailure
позволяет отличить изменение тарифа/провайдера от повторения временного запроса.

### Отдельный RPC для логов (лимит 10 блоков подтверждён)

В responseWETH_new_1795361584620.json пользователя, block 26471572,
logFailure сообщил HTTP 400, rpc_log_range_limit, providerMaxBlockRange=10,
requiredRangesForWindow=719. Это подтверждённый отказ по диапазону логов;
наличие архивного eth_call у этого endpoint не снимает лимит eth_getLogs.

Добавлена необязательная переменная среды:

```env
STAKEDAO_ETHEREUM_LOGS_RPC_URL=https://ethereum-rpc.publicnode.com
```

Основной STAKEDAO_ETHEREUM_RPC_URL можно оставить Alchemy. В GoLand обе
переменные задаются в той же Run Configuration → Environment variables,
откуда приложение уже читает основной RPC. После изменения нужен перезапуск.
PublicNode проверен для суточного окна; доступ к 7/30 дням зависит от его
политики. Вместо него можно указать свой RPC, который разрешает нужный диапазон.
Не менять 1500 на 10 для суточной истории без учёта сотен необходимых запросов.

`feeLogSource.go` открывает отдельный клиент **только при явном weethFeeDays**.
Он обслуживает logs и проверки идентичности их источника. Цены, состояния
контрактов, исторические eth_call, BOLD и discovery используют основной RPC.
Без новой переменной сохраняется прежний единственный источник.

У двух клиентов один sample budget и один ограничитель **5 RPC/сек, 300
попыток на diagnostic-запрос**. Дополнительный клиент не получает свой бюджет.
Проверяются chain ID и hash начального/конечного блоков относительно основного
источника; после чтения повторно проверяется конечный hash источника логов.
Клиент закрывается при успехе и при ошибке. Разные сети, отсутствующий блок,
несовпадение hash или недоступность источника оставляют оценку unavailable.
Уже рассчитанный APR сохраняется; скрытого переключения обратно на основной
RPC при отказе второго нет. API и ценовые fallback не добавлены.

В weethFees добавлены `logSource` (primary_rpc / configured_logs_rpc) и
`logSourceVerified`. Последний подтверждает идентичность сети/блоков, а не
полноту логов; полнота по-прежнему отражена в status/rangesDone/rangesTotal.
Ни URL, ни API-ключ второго источника в ответ не попадают.

При ручном обновлении нужны **четыре Go-файла** из новой версии:
StkDaoSRV.go, weethFeeModel.go, **новый feeLogSource.go** и dto/diagnostics.go.
Для схемы Swagger также обновляются docs/docs.go, swagger.json, swagger.yaml.
Старый набор из двух файлов больше не включает всю реализацию.

Подтверждённая weETH `yieldBreakdown.trading.apy` остаётся null с причиной
unsupported_pool_model. Оценка имеет собственное имя и не выдаётся за эту APY.

## Полнота и ограничения нагрузки

Переиспользован timestamp-поиск границы из tradingHistory.go. Логи читаются
после начального snapshot до конечного включительно: (startBlock,endBlock].
Диапазоны по 1500 блоков, три worker, общий лимит sampleRPC=5 запросов/с,
300 RPC на весь diagnostic. Лимитер защищён mutex для конкурентных reservation.
Максимум 160 диапазонов и 50000 событий на оценку; обхода лимитов нет.

Ошибка любого диапазона, неверный адрес/topic/index/сумма, removed log или
конфликтующая копия лога делают всю оценку unavailable. Одинаковые дубликаты
учитываются один раз. Крупные корректные swaps не отбрасываются по USD-порогу;
decimals не угадываются. Проверяются hash начального блока повторно и hash
конечного блока существующей проверкой ответа. URL/ключи не попадают в reasons.
Успешная пустая история даёт 0 только для **оценки swap fees**.

Никакой новой БД/миграции и кэша log-истории нет. Повторный запрос заново читает
диапазон: перед регулярной загрузкой 7/30 дней нужен отдельный инкрементальный
индекс с cursor и политикой reorg. Для точной модели следующий шаг — источник
реальных fee/admin сумм операций и исторического капитала, как описано в
[исследовании модели](BOLD_ROUTE_AND_WEETH_MODEL_RU.md).

## BOLD/USDC: что добавлено в код

Обычная диагностика этого gauge теперь автоматически добавляет
`diagnostics.rewardRoute`. Дополнительное сканирование logs для неё не запускается.

```text
gauges = 0x07a01471fA544D9C6531B631E6A96A79a9AD05E9
weethFeeDays = не задавать
```

`rewardRoute.go` проверяет reviewed Receiver/Legacy/Router, соответствие vault,
Controller, gauge receiver Locker, регистрацию BOLD и его distributor. Только
совпадение проверенных связей даёт status=configured. Смена адреса/нечитаемый
getter не приводит к угадыванию версии. Конфигурация не доказывает путь каждой
будущей execute-транзакции.

Показываются vaultDistributionStatus, BOLD-балансы двух Receiver,
claimableLocker (накопленная сумма, не rate) и Merkle target AllMight. AllMight
не называется CampaignRewardsDistributor: это разные роли.

Читаются последние восемь incentives проверенного CampaignRewardsDistributor.
Выбираются только точные gauge+BOLD, fromChainId=1, sender/manager=AllMight;
сверяются duration и start/end. Возвращаются funding-period statuses,
campaignsExamined и **campaignCoverage=last_8_ids_not_complete_registry**.
Отсутствие записи в этой выборке не доказывает отсутствие кампаний. Истёкшее
финансирование не доказывает отсутствие неclaimed выплат. APR из amount/duration
не присваивается, Merkle-компонент остаётся неизвестным.

Дальше для BOLD нужны: полноценный индекс incentives/маршрутов, правила
распределения между позициями, данные root/proof, исключение повторного учёта
gauge→campaign→vault. [Подтверждённая транзакция и адреса](BOLD_ROUTE_AND_WEETH_MODEL_RU.md).

## Проверки

На mainnet PublicNode для weETH, block **26457038**: 63 RPC за **12.712 секунд**,
5/5 диапазонов за 86400 секунд, 63 swaps, оценка fee APR **0.1916828210%**.
APR vault **1.1609619174%** остаётся отдельным; APY=null. Ошибок RPC/полей нет.
[Полный JSON](../../../examples/stakedao-weeth-fee-estimate.json).

Для BOLD, block **26457045**: 83 RPC, route configured, vault distribution
not_started, найдено пять подходящих ретроспективных кампаний среди восьми
прочитанных. Это не доказательство полноты реестра или известного APR.
[Полный JSON](../../../examples/stakedao-bold-route-diagnostic.json).

Тесты добавлены в существующие тестовые файлы: независимая проверка f/(1-f),
admin share/rate, типизированные события и дубликаты, полное покрытие диапазона,
отказ истории без нулевой подстановки и утечки URL, запрет смешивания с APR/APY,
неверные параметры HTTP, смена route identity, фильтрация sender/period кампаний,
общий concurrent RPS/budget. Отдельные mainnet вызовы в обычных тестах не делаются.

Отдельный log-клиент проверен на mainnet block **26471611**: 5/5 диапазонов,
82 swaps, **67 RPC за 13.58 секунд**, logSource=configured_logs_rpc,
logSourceVerified=true. Оценка fee APR=**1.8094272884%**, отдельно от APR/APY.
В этом контрольном прогоне оба независимо открытых клиента использовали
PublicNode; личный Alchemy endpoint пользователя не запускался из среды.
[JSON](../../../examples/stakedao-weeth-two-rpc.json).
Тест с двумя разными fake RPC отдельно подтверждает, что источник, отвергающий
логи по лимиту 10 блоков, не получает log-запросы, когда задан второй источник.
Проверены общий budget, ошибки dial, неверная сеть/hash, изменение hash после
чтения, закрытие второго клиента и отсутствие его URL в JSON.
