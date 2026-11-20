package stakedao

import (
	dto "ABIevm/internal/service/stakedao/dto"
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"math/big"
)

// Reviewed receiver/router versions, source + transfer proof:
// BOLD_ROUTE_AND_WEETH_MODEL_RU.md. Configuration is not proof of a payout.
var boldRouteGauge = common.HexToAddress("0x07a01471fa544d9c6531b631e6a96a79a9ad05e9")
var boldRouteToken = common.HexToAddress("0x6440f144b7e50d6a8439336510312d2f54beb01d")
var boldRouteReceiver = common.HexToAddress("0xaee4d726c1ccdb82e79e7cf9922fa6f5b89320d3")
var boldRouteLegacy = common.HexToAddress("0x8e980f6d681a02a701d4ce5ef68f4786351a3d8f")
var boldRouteRouter = common.HexToAddress("0x9a207a85e372fcdac3014f945a65868f2a05ba12")
var boldRouteExecutor = common.HexToAddress("0xdbd24b092f686b12650ec1450e3a7138f714506c")
var boldCampaignDistributor = common.HexToAddress("0xd4898a378ea555595c4e7dbde722b134a3f346d1")

const rewardRouteABI = `[
{"type":"function","name":"rewardVault","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"rewardRouter","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"legacyRewardReceiver","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"PROTOCOL_CONTROLLER","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"merkleDistributor","inputs":[],"outputs":[{"type":"address"}]},
{"type":"function","name":"getRewardsDistributor","inputs":[{"type":"address"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"isRewardToken","inputs":[{"type":"address"}],"outputs":[{"type":"bool"}]},
{"type":"function","name":"rewards_receiver","inputs":[{"type":"address"}],"outputs":[{"type":"address"}]},
{"type":"function","name":"claimable_reward","inputs":[{"type":"address"},{"type":"address"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"balanceOf","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"nbIncentives","inputs":[],"outputs":[{"type":"uint256"}]},
{"type":"function","name":"incentives","inputs":[{"type":"uint256"}],"outputs":[{"type":"address"},{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"address"},{"type":"uint256"},{"type":"address"}]}
]`

func (s *StakeDAOService) addBoldRewardRoute(ctx context.Context, block *big.Int, pool *dto.StakeDAOPool) {
	if s.chainID != 1 || pool.GaugeAddress != boldRouteGauge {
		return
	}
	if pool.Diagnostics == nil {
		pool.Diagnostics = &dto.StakeDAOPoolDiagnostics{}
	}
	d := &dto.RewardRouteDiagnostic{Status: "unavailable", BlockNumber: block.Uint64(), Token: boldRouteToken, Receiver: pool.RewardReceiver, VaultDistributionStatus: "unavailable", Campaigns: []dto.RewardCampaignDiagnostic{}, CampaignCoverage: "not_read"}
	pool.Diagnostics.RewardRoute = d
	fail := func(reason string) { d.Reasons = []string{reason} }
	if pool.AssetAddress != common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e") || pool.RewardReceiver != boldRouteReceiver || pool.ProtocolID != "0xc715e373" {
		fail("unsupported_reward_route")
		return
	}
	if pool.Diagnostics.Primary == nil || pool.Diagnostics.Primary.Locker == (common.Address{}) {
		fail("locker_unavailable")
		return
	}
	locker := pool.Diagnostics.Primary.Locker
	reads := []snapshotRead{
		readContract(d.Receiver, rewardRouteABI, "rewardVault"), readContract(d.Receiver, rewardRouteABI, "legacyRewardReceiver"), readContract(d.Receiver, rewardRouteABI, "rewardRouter"),
		readContract(boldRouteGauge, rewardRouteABI, "rewards_receiver", locker), readContract(pool.VaultAddress, rewardRouteABI, "getRewardsDistributor", boldRouteToken),
		readContract(boldRouteLegacy, rewardRouteABI, "rewardVault"), readContract(boldRouteRouter, rewardRouteABI, "PROTOCOL_CONTROLLER"), readContract(boldRouteRouter, rewardRouteABI, "merkleDistributor"),
		readContract(pool.VaultAddress, rewardRouteABI, "isRewardToken", boldRouteToken),
	}
	data := s.snapshotCalls(ctx, block, reads...)
	addresses := make([]common.Address, 8)
	for i, r := range data[:8] {
		a, e := r.address()
		if e != nil {
			fail("reward_route_getter_unavailable")
			return
		}
		addresses[i] = a
	}
	if data[8].err != nil || len(data[8].values) != 1 {
		fail("reward_token_registration_unavailable")
		return
	}
	registered, ok := data[8].values[0].(bool)
	if !ok || !registered {
		fail("reward_token_not_registered")
		return
	}
	d.LegacyReceiver, d.Router, d.GaugeReceiver, d.VaultDistributor, d.MerkleTarget = addresses[1], addresses[2], addresses[3], addresses[4], addresses[7]
	if addresses[0] != pool.VaultAddress || addresses[1] != boldRouteLegacy || addresses[2] != boldRouteRouter || addresses[3] != d.Receiver || addresses[4] != boldRouteLegacy || addresses[5] != pool.VaultAddress || addresses[6] != s.controller || addresses[7] != boldRouteExecutor {
		fail("reward_route_identity_mismatch")
		return
	}
	d.Status = "configured"
	d.Reasons = []string{"campaign_allocation_rules_and_proofs_required", "configuration_does_not_prove_selected_execution_path", "upstream_and_campaign_rewards_must_not_be_double_counted"}
	for _, r := range pool.Rewards {
		if r.Token == boldRouteToken && r.Source == "vault-extra" {
			d.VaultDistributionStatus = r.DistributionStatus
		}
	}
	amounts := s.snapshotCalls(ctx, block,
		readContract(boldRouteToken, rewardRouteABI, "balanceOf", d.Receiver), readContract(boldRouteToken, rewardRouteABI, "balanceOf", d.LegacyReceiver),
		readContract(boldRouteGauge, rewardRouteABI, "claimable_reward", locker, boldRouteToken), readContract(boldCampaignDistributor, rewardRouteABI, "nbIncentives"))
	targets := []**string{&d.ReceiverBalance, &d.LegacyBalance, &d.ClaimableLocker}
	for i, target := range targets {
		n, err := amounts[i].unsigned()
		if err != nil {
			d.Status = "partial"
			d.Reasons = append(d.Reasons, "reward_route_balance_unavailable")
			continue
		}
		v := decimal.NewFromBigInt(n, -18).String()
		*target = &v
	}
	count, err := amounts[3].unsigned()
	if err != nil || !count.IsUint64() {
		d.Status = "partial"
		d.Reasons = append(d.Reasons, "campaign_count_unavailable")
		return
	}
	n := count.Uint64()
	first := uint64(0)
	if n > 8 {
		first = n - 8
	}
	reads = nil
	for i := first; i < n; i++ {
		reads = append(reads, readContract(boldCampaignDistributor, rewardRouteABI, "incentives", new(big.Int).SetUint64(i)))
	}
	d.CampaignCoverage = "last_8_ids_not_complete_registry"
	entries := s.snapshotCalls(ctx, block, reads...)
	for i, e := range entries {
		if e.err != nil {
			d.Status = "partial"
			d.Reasons = append(d.Reasons, "campaign_entry_unavailable")
			continue
		}
		campaign, match, valid := decodeBoldCampaign(e.values, first+uint64(i), s.snapshotTime)
		if !valid {
			d.Status = "partial"
			d.Reasons = append(d.Reasons, "invalid_campaign_entry")
			continue
		}
		d.CampaignsExamined++
		if match {
			d.Campaigns = append(d.Campaigns, campaign)
		}
	}
}

func decodeBoldCampaign(v []interface{}, id, now uint64) (dto.RewardCampaignDiagnostic, bool, bool) {
	d := dto.RewardCampaignDiagnostic{ID: id, Distributor: boldCampaignDistributor}
	if len(v) != 9 {
		return d, false, false
	}
	gauge, gOK := v[0].(common.Address)
	token, tOK := v[1].(common.Address)
	sender, sOK := v[6].(common.Address)
	manager, mOK := v[8].(common.Address)
	if !gOK || !tOK || !sOK || !mOK {
		return d, false, false
	}
	ns := make([]*big.Int, 5)
	for j, i := range []int{2, 3, 4, 5, 7} {
		n, ok := v[i].(*big.Int)
		if !ok || n == nil || n.Sign() < 0 {
			return d, false, false
		}
		ns[j] = n
	}
	duration, start, end, chain, amount := ns[0], ns[1], ns[2], ns[3], ns[4]
	if !duration.IsUint64() || duration.Sign() <= 0 || !start.IsUint64() || !end.IsUint64() || end.Cmp(start) < 0 || new(big.Int).Sub(end, start).Cmp(duration) != 0 {
		return d, false, false
	}
	if gauge != boldRouteGauge || token != boldRouteToken || sender != boldRouteExecutor || manager != boldRouteExecutor || chain.Cmp(big.NewInt(1)) != 0 {
		return d, false, true
	}
	d.Start, d.End, d.Amount = start.Uint64(), end.Uint64(), decimal.NewFromBigInt(amount, -18).String()
	d.Status = "active_funding_period"
	if d.End <= now {
		d.Status = "expired_funding_period"
	} else if d.Start > now {
		d.Status = "future_funding_period"
	}
	return d, true, true
}
