# Convex: обзор источников для расчёта StakeDAO

Найдены два полезных Ethereum-контракта: **PoolUtilities** для ставок наград
и арифметического расчёта APR, **CvxMining** для оценки новой эмиссии CVX.
Они помогают независимо проверить Convex-часть доходности, но не заменяют
USD-цены, учёт StakeDAO Locker/Sidecar и комиссии Accountant.

## Что удалось прочитать

Полный обзор сайта `https://docs.convexfinance.com/convexfinance` **не завершён**:
из этой среды сайт, sitemap и GitBook-адрес возвращают HTTP 403 на сетевом
доступе. Страницы самого сайта прочитать не удалось; официального экспорта
GitBook не найдено. Для завершения нужен PDF/Markdown-экспорт со всеми разделами.

Вместо недоступных страниц полностью прочитаны доступные официальные README:

| Репозиторий Convex | Что полезно для интеграции |
| --- | --- |
| [platform](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/README.md) | Curve LP, CRV/cvxCRV, CVX; разные получатели комиссий и наград. README содержит также исторические сведения запуска. |
| [sidechain-platform](https://github.com/convex-eth/sidechain-platform/blob/72ef3da1e27d2fd515368bbfbfd364e80e40fb04/README.md) | Другой механизм начисления наград вне mainnet; общего `rewardRate` нет. |
| [frax-cvx-platform](https://github.com/convex-eth/frax-cvx-platform/blob/main/README.md) | Отдельные пользовательские vault и veFXS; другая модель размещения. |
| [function_x](https://github.com/convex-eth/function_x/blob/main/README.md) | Отдельные пользовательские vault и veFXN. |
| [voting](https://github.com/convex-eth/voting/blob/main/README.md) | Реестр голосований и делегирование; веса голосов не равны APR. |
| [utilities](https://github.com/convex-eth/utilities/blob/main/README.md), [prisma](https://github.com/convex-eth/prisma/blob/main/README.md) | Только заголовки; содержательной документации в этих README нет. |

Дополнительно изучены восемь исходников mainnet: `PoolUtilities`, `CvxMining`,
`Booster`, `BaseRewardPool`, `VirtualBalanceRewardPool`, `Cvx`,
`StashTokenWrapper`, `CvxCrvUtilities`; официальный адресный JSON и ABI двух
новых вспомогательных контрактов на Etherscan. Это обзор перечисленных
источников, а не аудит всех контрактов или всех разделов GitBook.

Версии: `platform@4f22038387ca4ce014dec3d7f5781a8e28051c13`,
`sidechain-platform@72ef3da1e27d2fd515368bbfbfd364e80e40fb04`.
URL, размеры и SHA-256 прочитанных README, а также результаты чтений сохранены
в [examples/convex-reference-review.json](../../../examples/convex-reference-review.json).
Ссылки остальных репозиториев на `main` могут изменяться.

## Проверенные адреса Ethereum

| Контракт | Адрес | Подтверждение |
| --- | --- | --- |
| PoolUtilities | `0x5Fba69a794F395184b5760DAf1134028608e5Cd1` | Официальный адресный JSON, верифицированный исходник/ABI, чтение методов. |
| CvxMining | `0x3c75BFe6FbfDa3A94E7E7E8c2216AFc684dE5343` | Ссылка из PoolUtilities, верифицированный ABI, чтение конверсии. |
| Booster | `0xF403C135812408BFbE8713b5A23a04b3D48AAE31` | Getter PoolUtilities, `poolInfo`, текущие комиссии. |
| CVX | `0x4e3FBD56CD56c3e72c1403e103b45Db9da5B9D2B` | Исходники, чтение `totalSupply/maxSupply`. |

Источники:
[официальный адресный JSON](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts.json),
[PoolUtilities на Etherscan](https://etherscan.io/address/0x5Fba69a794F395184b5760DAf1134028608e5Cd1#readContract),
[CvxMining на Etherscan](https://etherscan.io/address/0x3c75BFe6FbfDa3A94E7E7E8c2216AFc684dE5343#readContract).
Исходник PoolUtilities из GitHub совпадает с верифицированным исходником Etherscan.

Ещё полезные адреса из официального JSON: Convex VoterProxy
`0x989AEb4d175e16225E39E87d0D97A3360524AD80`, cvxCRV
`0x62B9c7356A2Dc64a1969e19C23e4f579F9810Aa7`, cvxCRVRewards
`0x3Fe65692bfCD0e6CF84cB1E7d24108E434A7587e`, CVXRewards
`0xCF50b810E57Ac33B91dCF525C6ddd9881B139332`, vlCVX
`0x72a19342e8F1838460eBFCCEf09F6585e32db86E`, CvxCrvUtilities
`0xadd2F542f9FF06405Fabf8CaE4A74bD0FE29c673`.
В этом обзоре их перечисление подтверждено источником; отдельная проверка
развёрнутого кода каждого из них не проводилась.

Статический JSON содержит лишь 66 записей пулов и не содержит pid=182,
который существует в блокчейне. Полный актуальный список следует получать
через `Booster.poolLength()/poolInfo(pid)`, связи конкретного Sidecar —
проверять по его собственному pid и каноническому `poolInfo`.

## PoolUtilities: функция APR действительно есть

[Исходник PoolUtilities](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/PoolUtilities.sol)
предоставляет:

```solidity
rewardRates(uint256 pid) returns (address[] tokens, uint256[] rates)
apr(uint256 rate, uint256 priceOfReward, uint256 priceOfDeposit) returns (uint256)
```

`rewardRates` читает BaseRewardPool из `Booster.poolInfo(pid)` и возвращает:

1. CRV: активный `rewardRate`, нормированный на общий staked supply.
2. CVX: прогноз новой эмиссии из CRV через CvxMining.
3. Остальные потоки: активные ставки из `extraRewards(i)`.

Для CRV и extra при положительном supply:

```text
rateRawPerLp = floor(rewardRateRaw × 10^18 / stakedSupplyRaw)
```

Периоды проверяются по timestamp блока; helper использует `timestamp <= periodFinish`.
При истёкшем периоде соответствующая ставка нулевая. При **нулевом supply**
контракт пропускает деление: результат нельзя интерпретировать как обычную
доходность на один LP. В клиенте нужна отдельная проверка положительного supply.

Для pid >= 151 адрес extra-токена разворачивается через `wrapper.token()`.
Один токен может присутствовать в нескольких слотах: например, CVX как новая
эмиссия и как отдельная extra-награда. Идентификатор источника нужно сохранять.
Нельзя автоматически удалять дубликаты по адресу или прибавлять все слоты
к уже учтённым наградам StakeDAO vault.

`apr(...)` выполняет только арифметику:

```text
resultRaw = rate × 31536000 × rewardPrice / depositPrice
```

Цены caller передаёт сам, в одинаковой точности. При LP/reward с 18 decimals
и ставке helper результат — годовая доля с масштабом `10^18`:
`APR% = resultRaw / 10^18 × 100`. Например, проверенный вызов
`apr(1000000000, 200000000, 100000000)` вернул `63072000000000000`,
то есть **6.3072%**. Это искусственные входные данные, не APR реального пула.

Если decimals различаются, требуется нормализация:

```text
tokensPerSecondPerOneLp = rateRawPerLp × 10^(lpDecimals − rewardDecimals) / 10^18
APR% = tokensPerSecondPerOneLp × 31536000 × rewardUSD / lpUSD × 100
```

Helper не узнаёт USD-цены, не учитывает StakeDAO Accountant и не объединяет
доходность Locker с Sidecar. Готового полного APR StakeDAO из одного pid он
не возвращает. Его можно использовать как дополнительную диагностику
Convex-части, сохраняя наш расчёт фактической позиции Sidecar и комиссий StakeDAO.

## Чтение двух пулов на одном блоке

На блоке **26348670**, timestamp `1793882147`, выполнено 21 read-only RPC-чтение
через PublicNode, без GetBlock CU и без отправки транзакций.

| Пул | pid | LP | BaseRewardPool | CRV rate helper |
| --- | ---: | --- | --- | ---: |
| crvUSD/USDC | 182 | `0x4dece678ceceb27446b35c672dc7d61f30bad69e` | `0x44d8fab7cd8b7877d5f79974c2f501af6e65abba` | `5122580168` raw |
| FXS/ETH | 69 | `0x90244f43d548a4f8dfecfad91a193465b1fad6f7` | `0xaa0e8ef60babda02ef11c89a061d82b1d61a462c` | `0` |

Для pid=182 проверено равенство:

```text
floor(40573308640085120 × 10^18 / 7920482901851079219537010) = 5122580168
```

CRV-период активен (`periodFinish=1794094067`). CVX и два extra-слота в
результате helper имеют нулевые ставки; среди адресов CVX указан дважды.
Для pid=69 оба возвращённых слота CRV/CVX нулевые. Нулевая ставка награды
не означает нулевую стоимость LP или отсутствие торговой доходности Curve.
Сравнение с текущим UI на том же блоке в этот обзор не входило.

## CvxMining: новая эмиссия и накопленный CVX различаются

[Исходник CvxMining](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/interfaces/CvxMining.sol)
реализует `ConvertCrvToCvx(uint256)`:

```text
cliff = floor(CVX.totalSupply / reductionPerCliff)
если cliff >= totalCliffs: новая эмиссия = 0
иначе:
  amount = floor(crvAmount × (totalCliffs − cliff) / totalCliffs)
  новая эмиссия = min(amount, maxSupply − totalSupply)
```

На блоке проверки `CVX.totalSupply == CVX.maxSupply == 100000000 × 10^18`,
а `ConvertCrvToCvx(10^18)` возвращает 0. Это подтверждает отсутствие новой
cliff-эмиссии, но не отменяет накопленные CVX или выплаты из существующих запасов.
Наш ранее найденный баланс CVX на Sidecar не становится текущим APR, пока
не запущен фактический поток vault. Подробности — в
[аудите CVX](CVX_REWARD_AUDIT_RU.md).

## Комиссии: не удерживать Convex дважды

[Booster._earmarkRewards](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/Booster.sol)
сначала получает gross CRV из gauge, удерживает комиссии Convex, затем
переводит остаток в BaseRewardPool и вызывает `queueNewRewards`.
Следовательно, **BaseRewardPool.rewardRate уже сформирован после комиссий Convex**.

На блоке 26348670 прочитаны изменяемые параметры:

| Getter Booster | Значение / FEE_DENOMINATOR=10000 | Доля |
| --- | ---: | ---: |
| lockIncentive | 1000 | 10% |
| stakerIncentive | 450 | 4.5% |
| earmarkIncentive | 50 | 0.5% |
| platformFee | 200 | 2% |
| Сумма | 1700 | **17%** |

Treasury на этом блоке ненулевой. Конструктор в исходнике задаёт другие
начальные значения: для актуального расчёта следует читать getters,
а не фиксировать исторический процент из README или defaults исходника.

Эти 17% относятся к **Convex**, а не к Accountant StakeDAO. Наш текущий путь
от BaseRewardPool не должен повторно вычитать их. Комиссии Accountant применяются
отдельно согласно проверенной политике: protocol fee к Locker CRV, harvest fee
к соответствующим primary-потокам. Потоки vault-extra не получают повторное
удержание Accountant. Новая находка подтверждает эту часть текущего подхода.

## Сроки выплат и типы контрактов

- [BaseRewardPool](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/BaseRewardPool.sol)
  и [VirtualBalanceRewardPool](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/VirtualBalanceRewardPool.sol)
  распределяют награды периодами по 7 дней. `earned` ограничивает начисление
  концом периода. Сохранённый положительный rate после окончания не является
  действующей выплатой.
- `queuedRewards`, `donate` и условия `queueNewRewards` (`newRewardRatio=830`)
  могут оставить награды в очереди. Баланс/очередь/разовый harvest не являются
  скоростью годовой доходности.
- VirtualBalanceRewardPool использует balances/supply родительского deposit
  pool: это не ещё одна независимая позиция пользователя, которую нужно
  прибавлять к TVL.
- [StashTokenWrapper](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/StashTokenWrapper.sol)
  возвращает underlying через `token()`, а `decimals()` берёт у underlying.
  Нельзя оценивать wrapper как отдельный рыночный токен или всегда считать
  все extra-токены 18-decimal.
- Shutdown пула запрещает новые депозиты, но сам по себе не доказывает
  нулевой остаток LP. В Booster предусмотрен возврат LP при shutdown.

## Цены, торговая доходность и границы модели

Обнаруженный `apr` требует цену LP в USD. Он не заменяет наш ценовой сервис:
для LP стоимость можно определять по стоимости активов и `totalSupply`
с учётом конкретного типа Curve-пула; для ERC4626 — через конверсию shares
в underlying. `virtual_price` сам по себе не гарантирует USD-котировку.
Проверенный cache/oracle/NAV-путь остаётся нужен. Источник цены и блок должны
быть согласованы с данными наград; отсутствие цены нельзя заменять долларом.

Rewards APR и торговая APY Curve — разные компоненты. PoolUtilities считает
первую, а не доходность от изменения LP NAV/комиссий сделок. Статический snapshot
не даёт фактически реализованную торговую доходность за период. Поэтому находка
не обосновывает заполнение нашего `apy` или автоматическое сложное начисление.

Другие продукты Convex также нельзя смешивать с Curve LP:

- Convex VoterProxy и StakeDAO Locker — разные владельцы размещений.
  Глобальный stake Convex нельзя подставлять вместо позиции StakeDAO.
- cvxCRV выпускается при бессрочной блокировке CRV; обмен через рыночный пул
  не равен гарантированному погашению 1:1. Это не основание считать USD-цену
  cvxCRV равной цене CRV.
- CVX staking и vlCVX locking имеют собственные потоки дохода; их комиссии
  не прибавляются к APR произвольного Curve LP.
- [CvxCrvUtilities](https://github.com/convex-eth/platform/blob/4f22038387ca4ce014dec3d7f5781a8e28051c13/contracts/contracts/CvxCrvUtilities.sol)
  рассчитывает группы наград и пользовательские веса для cvxCRV wrapper.
  Это другая позиция. В нём есть специальные множители для нулевого supply,
  поэтому переносить его результат на обычные LP-vault нельзя.
- Frax/f(x) используют свои пользовательские vault. Наличие FXS в FXS/ETH
  не превращает этот Curve-пул в продукт Frax Convex.
- Sidechain ConvexRewardPool обновляет распределение по поступившим балансам
  при действиях пользователей и не имеет общего rewardRate. Для новой сети
  потребуются её источники наград и hooks, а не копирование mainnet ABI.
- Реестр голосований `getAddress(platform,type)` помогает искать governance
  контракты, но голосовой вес не равен реализованной эмиссии/доходности.
  Адрес развёрнутого реестра этим обзором не подтверждён.

## Что стоит добавить следующим шагом

1. В диагностике двух пулов опционально читать PoolUtilities.rewardRates(pid)
   на том же блоке и сравнивать CRV со ставкой BaseRewardPool и собственным
   размещением Sidecar. Расхождение выводить с единицами и источниками.
2. Сохранять раздельно текущую выплату, прогноз mint и pending balance;
   не превращать CVX на Sidecar в текущую доходность до распределения vault.
3. Для точного сравнения с UI отдельно согласовать состав показателя:
   rewards APR, торговая доходность, комиссии и момент ценового среза.

В рамках этого обзора сохранены только документация и результаты чтений.
Код расчётов, endpoints, ABI приложения, БД и другие платформы не изменены.
