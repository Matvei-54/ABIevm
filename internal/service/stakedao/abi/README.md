# Происхождение ABI

JSON содержат только необходимые view-методы из верифицированных ABI Etherscan,
полученных при выполнении шага 5. Порядок и типы выходов сохранены.

| Файл | Верифицированный контракт |
| --- | --- |
| accountant-v2.json | [Accountant](https://etherscan.io/address/0x93b4B9bd266fFA8AF68e39EDFa8cFe2A62011Ce0#code) |
| vault-v2.json | [RewardVault implementation](https://etherscan.io/address/0x74D8dd40118B13B210D0a1639141cE4458CAe0c0#code) |
| strategy-v2.json | [CurveStrategy](https://etherscan.io/address/0xcE37598c3b26eCbdF13356f9E1FFf2Cd36EE602a#code) |
| sidecar-v2.json | Общие getters фактической [ConvexSidecar implementation 0x66c3…](https://etherscan.io/address/0x66c3ce4718A39d44CE2430eB3E8B8d43c18bA1fa#code) и [версии 0xef7c…](https://etherscan.io/address/0xef7cfFDeF21538B63Cc541bf206BB382d922A216#code) |

`curve-gauge.json` и `curve-controller.json` — подмножества существующих
`contracts/curve/LiquidityGauge.json` и `CurveGaugeController.json`.
Сигнатура allocator подтверждена `IAllocator.sol` в верифицированном исходном
коде CurveStrategy. Методы Convex base pool подтверждены
[официальным BaseRewardPool.sol](https://github.com/convex-eth/platform/blob/main/contracts/contracts/BaseRewardPool.sol).

Версия `0x66c3…`, используемая Sidecar crvUSD/USDC
`0xbdf7f41e0ffeaa220fa49c4e89490cc6b1d2a961`, не имеет `gauge()`.
В `0xef7c…` этот getter есть. Обе версии имеют `asset`, `rewardReceiver`,
`PROTOCOL_CONTROLLER`, `BOOSTER`, `pid`, `baseRewardPool`, `ACCOUNTANT`,
`REWARD_TOKEN`, `balanceOf`. Рабочий subset больше не зависит от `gauge()`.

Связи проверяются через текущую регистрацию Controller и канонический Ethereum
[Convex Booster](https://etherscan.io/address/0xF403C135812408BFbE8713b5A23a04b3D48AAE31#code).
`poolInfo(pid)` возвращает `(lpToken, token, gauge, crvRewards, stash, shutdown)`;
сверяются LP, Curve gauge и адрес baseRewardPool. Сигнатура подтверждена
`@stake-dao/interfaces/src/interfaces/convex/IBooster.sol` в верифицированных
исходниках обеих версий Sidecar. Ошибка любого чтения/связи оставляет primary APR
недоступным; отсутствующий getter не заменяется нулём и произвольный Booster
не принимается.

Проверены `RewardVault.depositRewards/_rewardPerToken`,
`CurveStrategy._checkpointRewards`, `Strategy._harvest` и
`Accountant.checkpoint`: масштаб extra rates равен `1e18`, протокольные fees
относятся к Locker, harvest fees в CHECKPOINT — ко всему primary amount.

Адреса в таблице служат источниками интерфейсов, не списком пулов.
Приложение получает адреса через события/актуальное состояние Controller.
В случае несовместимого интерфейса возвращается partial/unavailable APR;
не выполняется угадывание альтернативного ABI или подстановка нулевых rates.
