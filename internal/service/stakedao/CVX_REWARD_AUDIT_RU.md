# Аудит CRV/CVX/extra rewards: crvUSD/USDC

Проверка Ethereum на блоке **26334677**, timestamp `1793713751`.
Связи контрактов, балансы и периоды прочитаны на одном блоке через PublicNode:
39 RPC-запросов, без GetBlock CU. Контракты не изменялись, harvest/distribution
транзакции не отправлялись. Машиночитаемые результаты:
[examples/stakedao-cvx-audit.json](../../../examples/stakedao-cvx-audit.json).

## Проверенный путь наград

| Контракт | Адрес |
| --- | --- |
| RewardVault | `0xd4467fbcbd3511112d2fd1af667e745d4987c8eb` |
| ConvexSidecar | `0xbdf7f41e0ffeaa220fa49c4e89490cc6b1d2a961` |
| RewardReceiver | `0x98f38dddea5c07bb934e8c9211b93effc17b57bf` |
| Convex BaseRewardPool | `0x44d8fab7cd8b7877d5f79974c2f501af6e65abba` |

`receiver.rewardVault()` и `sidecar.rewardReceiver()` подтвердили эти связи.
Верифицированные исходники показывают два разных пути:

```text
CRV: BaseRewardPool → Sidecar._claim() → Accountant → вкладчики vault
CVX/extra: Sidecar.claimExtraRewards() → RewardReceiver
           → distributeRewards() → vault.depositRewards() → поток вкладчикам
```

`Sidecar._claim()` получает CRV и переводит его Accountant. CVX, появившийся
при Convex claim, может оставаться на Sidecar. `claimExtraRewards()` переводит
весь CVX и дополнительные токены в Receiver. Если CVX также входит в список
Convex extra tokens, повторный проход видит уже нулевой баланс CVX: одна сумма
не отправляется дважды.

`RewardReceiver` переводит только зарегистрированные vault-токены, только если
он является их distributor, и ждёт окончания уже идущего периода распределения.
`RewardVault.depositRewards` запускает линейное распределение на **604800 секунд**
(7 дней), сохраняет rate с дополнительным масштабом `1e18`. Его нормализация:
`tokens/sec = rawRewardRate / 10^(18 + tokenDecimals)`.

`CurveStrategy.claimExtraRewards(gauge)` в проверенной версии вызывает только
получение дополнительных наград Curve Locker. Он не вызывает автоматически
`Sidecar.claimExtraRewards()` и последующее распределение Receiver. Само наличие
этого метода стратегии не означает, что накопленный CVX уже попал в vault.

## Что обнаружено на блоке проверки

| Показатель CVX | Значение |
| --- | ---: |
| Баланс Sidecar | 296.453034313470865223 CVX |
| Баланс Receiver | 0 CVX |
| Баланс RewardVault | 0 CVX |
| Vault periodFinish / rawRewardRate | 0 / 0 |
| Статус распределения vault | `not_started` |

Эти 296.453 CVX — накопленный баланс на Sidecar, не текущий rate выплаты vault.
Включать его в текущий APR как годовую эмиссию или автоматически делить на семь
дней было бы необоснованно: период ещё не запускался. При реальном запуске
поток автоматически читается существующим `vault-extra` по rate/periodFinish.

Дополнительно проверены Convex virtual reward pools:

- CVX: `0xac183f7cd62d5b04fa40362eb67249a80339541a`, wrapper
  `0xc8dc88896aac1a94aa42b89d65aa5fd4984cb71d` → canonical CVX.
  `periodFinish=1686098747` уже истёк, `earned(sidecar)=0`.
- YB: `0x2c657d22b97f1678f9e6f3a19014914083ec615d`, wrapper
  `0x6c29412588efa1ff1bc74c0ea8d0451518798baf` → canonical YB.
  `periodFinish=0`, `rewardRate=0`, `earned(sidecar)=0`.
  Балансы YB на Sidecar/Receiver/vault нулевые; в vault YB не зарегистрирован.

`CVX.totalSupply()` и `CVX.maxSupply()` равны 100000000 CVX. Это исключает новую
cliff-эмиссию CVX на момент проверки, но не отменяет существующие CVX-балансы или
возможное распределение заранее созданных резервов. Текущего активного
CVX/YB-потока, пропущенного нашим расчётом, на этом блоке не обнаружено.

Параметры Accountant: protocol fee **16.5%**, harvest fee **0.025%**, POLICY=0
(CHECKPOINT). Protocol fee относится к primary Locker CRV; harvest fee —
к primary Locker+Sidecar CRV. Дополнительные потоки vault считаются по его rate
и не получают повторного удержания этих комиссий. Отдельный гипотетический
Convex CVX APR к уже учтённому `vault-extra` не добавляется.

## Сопоставление с опубликованным API StakeDAO

Live UI/Hub из среды проверки недоступен (HTTP 403). Использован опубликованный
[Ethereum V2 JSON](https://github.com/stake-dao/api/blob/9b038fcf46de16aeee8f01b714c4dc18f5f99632/api/strategies/v2/curve/1.json)
в фиксированной версии `9b038fcf46de16aeee8f01b714c4dc18f5f99632`.
В нём нет номера snapshot-блока. Это внешний контроль состава доходности,
а не независимая проверка точного совпадения на одном блоке.

Наш diagnostic endpoint на блоке **26334719**: 65 RPC-попыток за 12.929 секунды,
без ошибок полей; оба пула имеют `aprStatus=available`.

| Пул | Наш rewards APR | CRV APR в API | Trading Fees APY в API | Итог в API |
| --- | ---: | ---: | ---: | ---: |
| crvUSD/USDC | 4.2283919% | 4.2140312% | 1.83% | 6.0440312% |
| FXS/ETH | 0% | 0% | 0.97% | 0.97% |

Разница CRV APR crvUSD/USDC составляет около 0.01436 процентного пункта.
Блоки, балансы и ценовые срезы не синхронизированы; точную причину остаточной
разницы не утверждаем. Методика внутреннего Hub не открыта этим сравнением.
API суммирует отображаемые trading APY и reward APR; в нашем `aprScope=rewards`
базовая торговая доходность не включена, `apy` остаётся null.

## Изменение ответа приложения

В подробные `rewards` добавлен `distributionStatus` для `vault-extra`:
`active`, `not_started`, `expired`, `zero_rate`, `unavailable`.
Нечитаемые данные остаются неизвестными, а не подтверждённым нулём. Статус
распределения не заменяет общую проверку полноты APR и наличие USD-цен.
Краткая строка пула по-прежнему содержит только шесть полей.

Тестами проверены состояния периода и положительный CVX-поток: он учитывается
один раз по rate vault, с правильными decimals и без повторной комиссии
Accountant. HTTP-источники в расчёт цен или APR приложения не добавлены.

Исходники контрактов: [Sidecar](https://etherscan.io/address/0xbdf7f41e0ffeaa220fa49c4e89490cc6b1d2a961#code),
[Receiver](https://etherscan.io/address/0x98f38dddea5c07bb934e8c9211b93effc17b57bf#code),
[Vault](https://etherscan.io/address/0xd4467fbcbd3511112d2fd1af667e745d4987c8eb#code),
[Accountant](https://etherscan.io/address/0x93b4b9bd266ffa8af68e39edfa8cfe2a62011ce0#code),
[CurveStrategy](https://etherscan.io/address/0xce37598c3b26ecbdf13356f9e1fff2cd36ee602a#code).
