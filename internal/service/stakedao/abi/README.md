# Происхождение ABI

JSON содержат только необходимые view-методы из верифицированных ABI Etherscan,
полученных при выполнении шага 5. Порядок и типы выходов сохранены.

| Файл | Верифицированный контракт |
| --- | --- |
| accountant-v2.json | [Accountant](https://etherscan.io/address/0x93b4B9bd266fFA8AF68e39EDFa8cFe2A62011Ce0#code) |
| vault-v2.json | [RewardVault implementation](https://etherscan.io/address/0x74D8dd40118B13B210D0a1639141cE4458CAe0c0#code) |
| strategy-v2.json | [CurveStrategy](https://etherscan.io/address/0xcE37598c3b26eCbdF13356f9E1FFf2Cd36EE602a#code) |
| sidecar-v2.json | [ConvexSidecar implementation](https://etherscan.io/address/0xef7cfFDeF21538B63Cc541bf206BB382d922A216#code) |

`curve-gauge.json` и `curve-controller.json` — подмножества существующих
`contracts/curve/LiquidityGauge.json` и `CurveGaugeController.json`.
Сигнатура allocator подтверждена `IAllocator.sol` в верифицированном исходном
коде CurveStrategy. Методы Convex base pool подтверждены
[официальным BaseRewardPool.sol](https://github.com/convex-eth/platform/blob/main/contracts/contracts/BaseRewardPool.sol).

Проверены `RewardVault.depositRewards/_rewardPerToken`,
`CurveStrategy._checkpointRewards`, `Strategy._harvest` и
`Accountant.checkpoint`: масштаб extra rates равен `1e18`, протокольные fees
относятся к Locker, harvest fees в CHECKPOINT — ко всему primary amount.

Адреса в таблице служат источниками интерфейсов, не списком пулов.
Приложение получает адреса через события/актуальное состояние Controller.
В случае несовместимого интерфейса возвращается partial/unavailable APR;
не выполняется угадывание альтернативного ABI или подстановка нулевых rates.
