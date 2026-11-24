package stakedao

// Isolated BOLD/USDC research module. Nothing in the service/HTTP path calls it.
// Usage: NewBOLDCampaignInspector(stateRPC, logsRPC, nil).Inspect(ctx, from, block).
// Both clients can be *ethclient.Client; caller owns their lifetime. State calls
// require historical eth_call, log calls need the requested eth_getLogs range.
// No API, database, transactions, background jobs or automatic RPC fallback.
//
// Sources: verified CampaignRewardsDistributor + UniversalRewardDistributor at
// https://etherscan.io/address/0xd4898a378ea555595c4e7dbde722b134a3f346d1#code
// and RewardRouter at boldRouteRouter (BOLD_ROUTE_AND_WEETH_MODEL_RU.md).
// A campaign records funding, not the vault's entitlement. The global Merkle
// leaf is (account,reward,cumulativeClaimable), without campaign ID or gauge.

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

const boldCampaignResearchABI = `[
{"type":"function","name":"root","inputs":[],"outputs":[{"type":"bytes32"}]},
{"type":"function","name":"ipfsHash","inputs":[],"outputs":[{"type":"bytes32"}]},
{"type":"event","name":"RewardsPulledToMerkle","anonymous":false,"inputs":[{"name":"gauge","type":"address","indexed":true},{"name":"token","type":"address","indexed":true},{"name":"amount","type":"uint128","indexed":false}]},
{"type":"event","name":"IncentiveAdded","anonymous":false,"inputs":[{"name":"id","type":"uint256","indexed":true},{"name":"gauge","type":"address","indexed":true},{"name":"reward","type":"address","indexed":true},{"name":"manager","type":"address","indexed":false},{"name":"duration","type":"uint256","indexed":false},{"name":"start","type":"uint256","indexed":false},{"name":"end","type":"uint256","indexed":false},{"name":"amount","type":"uint256","indexed":false}]}
]`

const (
	boldResearchMaxRanges    = 160
	boldResearchMaxEvents    = 20000
	boldResearchMaxCampaigns = 64
	boldResearchMaxReceipts  = 32
	boldResearchMaxRPC       = 300
)

type BOLDCampaignStateReader interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}

type BOLDCampaignLogReader interface {
	ChainID(context.Context) (*big.Int, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
	FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error)
}

// BOLDAllocationResolver is a TRUSTED boundary for a future reviewed allocation
// model, not an arbitrary price/percentage provider. Implementations must prove
// vault eligibility, campaign attribution, net fees, absence of duplicate
// upstream/vault payouts, and the full-period capital integral. A Merkle proof
// alone does NOT establish those facts. Nil deliberately leaves APR unknown.
type BOLDAllocationResolver interface {
	ResolveBOLDAllocation(context.Context, BOLDCampaignReport, BOLDCampaignRecord) (BOLDAllocationEvidence, error)
}

// Evidence values are supplied by the reviewed resolver, never inferred from
// campaign funding. Snapshot-price conversion yields a historical ESTIMATE.
type BOLDAllocationEvidence struct {
	CampaignID         uint64         `json:"campaignId"`
	SnapshotHash       common.Hash    `json:"snapshotHash"`
	Vault              common.Address `json:"vault"`
	Start              uint64         `json:"start"`
	End                uint64         `json:"end"`
	NetRewardRaw       string         `json:"netRewardRaw"`
	RewardUnitUSD      string         `json:"rewardUnitUsd"`
	CapitalUSDSeconds  string         `json:"capitalUsdSeconds"` // integral of eligible vault capital, USD * seconds
	RulesReference     string         `json:"rulesReference"`
	ValuationReference string         `json:"valuationReference"`
	EvidenceHash       common.Hash    `json:"evidenceHash"`
}

type BOLDCampaignRecord struct {
	ID               uint64                  `json:"id"`
	Start            uint64                  `json:"start"`
	End              uint64                  `json:"end"`
	AmountRaw        string                  `json:"amountRaw"`
	Amount           string                  `json:"amount"`
	PeriodStatus     string                  `json:"periodStatus"`
	CreationBlock    uint64                  `json:"creationBlock"`
	TransactionHash  common.Hash             `json:"transactionHash"`
	LogIndex         uint                    `json:"logIndex"`
	FundingStatus    string                  `json:"fundingStatus"`
	AllocationStatus string                  `json:"allocationStatus"`
	Reasons          []string                `json:"reasons,omitempty"`
	HistoricalAPR    *string                 `json:"historicalApr"`
	Allocation       *BOLDAllocationEvidence `json:"allocation,omitempty"`
}

type BOLDCampaignReport struct {
	Status         string               `json:"status"`
	BlockNumber    uint64               `json:"blockNumber"`
	BlockTimestamp uint64               `json:"blockTimestamp"`
	BlockHash      common.Hash          `json:"blockHash"`
	FromBlock      uint64               `json:"fromBlock"`
	Coverage       string               `json:"coverage"` // creation blocks, NOT full reward periods
	ScanComplete   bool                 `json:"scanComplete"`
	RPCRequests    int                  `json:"rpcRequests"`
	Vault          common.Address       `json:"vault"`
	Gauge          common.Address       `json:"gauge"`
	Token          common.Address       `json:"token"`
	Distributor    common.Address       `json:"distributor"`
	Root           common.Hash          `json:"root"`
	IPFSHash       common.Hash          `json:"ipfsHash"`
	Campaigns      []BOLDCampaignRecord `json:"campaigns"`
	CurrentAPR     *string              `json:"currentApr"` // historical observations are never current APR
	Reasons        []string             `json:"reasons"`
}

type BOLDCampaignInspector struct {
	state      BOLDCampaignStateReader
	logs       BOLDCampaignLogReader
	allocation BOLDAllocationResolver
}

func NewBOLDCampaignInspector(state BOLDCampaignStateReader, logs BOLDCampaignLogReader, allocation BOLDAllocationResolver) *BOLDCampaignInspector {
	return &BOLDCampaignInspector{state: state, logs: logs, allocation: allocation}
}

// Inspect performs tasks 1-3 independently of StakeDAOService. It scans only the
// inclusive [from, block] creation window, in <=1500-block chunks; never assumes
// the newest 8 IDs form a full registry. block must be explicit and nonzero.
// An incomplete scan/provider failure returns a safe error, not a zero return.
// 300 calls and 5 calls/second are shared by both injected clients for this run.
// This budget covers the built-in reader; a future resolver must separately
// honour the application's RPC limits if it performs additional network reads.
func (a *BOLDCampaignInspector) Inspect(ctx context.Context, from, block uint64) (*BOLDCampaignReport, error) {
	if a == nil || a.state == nil || a.logs == nil || from == 0 || block < from || (block-from)/1500+1 > boldResearchMaxRanges {
		return nil, errors.New("bold_campaign_invalid_request")
	}
	r := &BOLDCampaignReport{Status: "unavailable", FromBlock: from, BlockNumber: block,
		Coverage: "campaign_creations_in_requested_blocks_only", Vault: common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"),
		Gauge: boldRouteGauge, Token: boldRouteToken, Distributor: boldCampaignDistributor, Campaigns: []BOLDCampaignRecord{},
		Reasons: []string{"campaign_registry_not_global", "historical_funding_is_not_current_apr"}}
	s := &boldResearchRun{owner: a, report: r}
	for _, readChain := range []func(context.Context) (*big.Int, error){a.state.ChainID, a.logs.ChainID} {
		if err := s.reserve(ctx); err != nil {
			return nil, err
		}
		chain, err := readChain(ctx)
		if err != nil || chain == nil || chain.Cmp(big.NewInt(1)) != 0 {
			return nil, errors.New("bold_campaign_chain_mismatch_or_unavailable")
		}
	}
	first, err := s.header(ctx, a.state.HeaderByNumber, from)
	if err != nil {
		return nil, err
	}
	end, err := s.header(ctx, a.state.HeaderByNumber, block)
	if err != nil {
		return nil, err
	}
	r.BlockTimestamp, r.BlockHash = end.Time, end.Hash()
	for _, h := range []*types.Header{first, end} {
		other, err := s.header(ctx, a.logs.HeaderByNumber, h.Number.Uint64())
		if err != nil || other.Hash() != h.Hash() {
			return nil, errors.New("bold_campaign_log_source_mismatch")
		}
	}
	if err := s.readRoute(ctx); err != nil {
		return nil, err
	}
	parsed, _ := abi.JSON(strings.NewReader(boldCampaignResearchABI))
	added := parsed.Events["IncentiveAdded"]
	pulled := parsed.Events["RewardsPulledToMerkle"]
	entries := map[uint64]types.Log{}
	seen := map[string]types.Log{}
	for start := from; ; {
		through := start + min(uint64(1499), block-start)
		if err := s.reserve(ctx); err != nil {
			return nil, err
		}
		logs, err := a.logs.FilterLogs(ctx, ethereum.FilterQuery{FromBlock: new(big.Int).SetUint64(start), ToBlock: new(big.Int).SetUint64(through),
			Addresses: []common.Address{boldRouteRouter, boldCampaignDistributor}, Topics: [][]common.Hash{{added.ID, pulled.ID}}})
		if err != nil {
			return nil, errors.New("bold_campaign_log_history_unavailable")
		}
		if len(logs) > boldResearchMaxEvents-len(seen) {
			return nil, errors.New("bold_campaign_event_limit")
		}
		for _, l := range logs {
			if err := boldResearchValidateLog(l, start, through); err != nil {
				return nil, err
			}
			key := l.TxHash.Hex() + ":" + new(big.Int).SetUint64(uint64(l.Index)).String()
			if old, exists := seen[key]; exists {
				if !boldResearchSameLog(old, l) {
					return nil, errors.New("bold_campaign_conflicting_logs")
				}
				continue
			}
			seen[key] = l
			if l.Address == boldCampaignDistributor && l.Topics[0] == added.ID {
				id, match, err := boldResearchCampaignEvent(l, added)
				if err != nil {
					return nil, err
				}
				if !match {
					continue
				}
				if id >= s.count {
					return nil, errors.New("bold_campaign_id_outside_snapshot_registry")
				}
				if _, duplicate := entries[id]; duplicate {
					return nil, errors.New("bold_campaign_duplicate_id")
				}
				entries[id] = l
				if len(entries) > boldResearchMaxCampaigns {
					return nil, errors.New("bold_campaign_count_limit")
				}
			} else if l.Address != boldRouteRouter || l.Topics[0] != pulled.ID {
				return nil, errors.New("bold_campaign_unexpected_event")
			}
		}
		if through == block {
			break
		}
		start = through + 1
	}
	ids := make([]uint64, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for start := 0; start < len(ids); start += 24 {
		reads := []snapshotRead{}
		through := min(start+24, len(ids))
		for _, id := range ids[start:through] {
			reads = append(reads, readContract(boldCampaignDistributor, rewardRouteABI, "incentives", new(big.Int).SetUint64(id)))
		}
		values, err := s.batch(ctx, reads)
		if err != nil {
			return nil, err
		}
		for j, v := range values {
			id := ids[start+j]
			c, match, valid := decodeBoldCampaign(v, id, end.Time)
			if !valid {
				return nil, errors.New("bold_campaign_invalid_getter")
			}
			if !match {
				continue
			} // excludes other senders/chains, even same gauge/token
			l := entries[id]
			if err := boldResearchCompareEvent(l, added, v); err != nil {
				return nil, err
			}
			r.Campaigns = append(r.Campaigns, BOLDCampaignRecord{ID: id, Start: c.Start, End: c.End, Amount: c.Amount,
				AmountRaw: v[7].(*big.Int).String(), PeriodStatus: c.Status, CreationBlock: l.BlockNumber, TransactionHash: l.TxHash,
				LogIndex: l.Index, FundingStatus: "unavailable", AllocationStatus: "unavailable", Reasons: []string{"allocation_rules_unverified"}})
		}
	}
	// Full receipts prove the funding route and conservation, not just two events
	// that happen to share a transaction hash. Ambiguous multi-pull txs fail closed.
	verifiedEntries := make(map[uint64]types.Log, len(r.Campaigns))
	for _, c := range r.Campaigns {
		verifiedEntries[c.ID] = entries[c.ID]
	}
	receipts := map[common.Hash]*types.Receipt{}
	for i := range r.Campaigns {
		c := &r.Campaigns[i]
		receipt := receipts[c.TransactionHash]
		if receipt == nil {
			if len(receipts) >= boldResearchMaxReceipts {
				return nil, errors.New("bold_campaign_receipt_limit")
			}
			if err := s.reserve(ctx); err != nil {
				return nil, err
			}
			receipt, err = a.state.TransactionReceipt(ctx, c.TransactionHash)
			if err != nil || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful || receipt.TxHash != c.TransactionHash || receipt.BlockNumber == nil || !receipt.BlockNumber.IsUint64() || receipt.BlockNumber.Uint64() != c.CreationBlock {
				return nil, errors.New("bold_campaign_receipt_unavailable_or_invalid")
			}
			h, err := s.header(ctx, a.state.HeaderByNumber, c.CreationBlock)
			if err != nil || h.Hash() != receipt.BlockHash {
				return nil, errors.New("bold_campaign_receipt_block_mismatch")
			}
			if err := boldResearchVerifyFunding(receipt, verifiedEntries, parsed); err != nil {
				return nil, err
			}
			receipts[c.TransactionHash] = receipt
		}
		c.FundingStatus = "verified_router_campaign_funding"
	}
	// Recheck boundary hashes on BOTH sources after the complete scan/getters.
	for _, readHeader := range []func(context.Context, *big.Int) (*types.Header, error){a.state.HeaderByNumber, a.logs.HeaderByNumber} {
		for _, before := range []*types.Header{first, end} {
			after, err := s.header(ctx, readHeader, before.Number.Uint64())
			if err != nil || after.Hash() != before.Hash() {
				return nil, errors.New("bold_campaign_snapshot_changed")
			}
		}
	}
	r.ScanComplete, r.Status = true, "funding_verified_allocation_unavailable"
	if len(r.Campaigns) == 0 {
		r.Status = "no_verified_campaigns_in_creation_window"
		r.Reasons = append(r.Reasons, "empty_creation_window_does_not_prove_zero_rewards")
	}
	for i := range r.Campaigns {
		c := &r.Campaigns[i]
		if a.allocation == nil {
			continue
		}
		// Pass no caller-mutable report slices to the trusted resolver.
		view := *r
		view.Campaigns = nil
		view.Reasons = append([]string(nil), r.Reasons...)
		evidence, err := a.allocation.ResolveBOLDAllocation(ctx, view, *c)
		if err != nil {
			c.Reasons = []string{"allocation_evidence_unavailable"}
			continue
		}
		apr, err := CalculateBOLDHistoricalAPR(*r, *c, evidence)
		if err != nil {
			c.Reasons = []string{err.Error()}
			continue
		}
		c.Allocation, c.HistoricalAPR, c.AllocationStatus = &evidence, &apr, "historical_estimate_available"
		c.Reasons = nil
	}
	for _, c := range r.Campaigns {
		if c.HistoricalAPR != nil {
			r.Status = "funding_verified_with_historical_estimates"
			break
		}
	}
	if a.allocation != nil {
		checked, err := s.header(ctx, a.state.HeaderByNumber, block)
		if err != nil || checked.Hash() != r.BlockHash {
			return nil, errors.New("bold_campaign_snapshot_changed")
		}
	}
	return r, nil
}

// CalculateBOLDHistoricalAPR consumes EXTERNALLY VERIFIED allocation evidence.
// It checks binding, units, period, funding cap and provenance, but cannot infer
// or prove allocation rules from caller-supplied strings. Never call it with
// total campaign funding substituted for a vault's net allocation. No addition
// to vault APR/APY or aggregation with gaugeExtra is performed.
func CalculateBOLDHistoricalAPR(r BOLDCampaignReport, c BOLDCampaignRecord, e BOLDAllocationEvidence) (string, error) {
	if !r.ScanComplete || r.BlockHash == (common.Hash{}) || c.FundingStatus != "verified_router_campaign_funding" ||
		r.Gauge != boldRouteGauge || r.Token != boldRouteToken || r.Distributor != boldCampaignDistributor || r.Vault != common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad") ||
		e.SnapshotHash != r.BlockHash || e.CampaignID != c.ID || e.Vault != r.Vault || e.Start != c.Start || e.End != c.End ||
		e.EvidenceHash == (common.Hash{}) || strings.TrimSpace(e.RulesReference) == "" || strings.TrimSpace(e.ValuationReference) == "" {
		return "", errors.New("allocation_evidence_unverified_or_mismatched")
	}
	if e.End <= e.Start || e.End > r.BlockTimestamp {
		return "", errors.New("allocation_period_not_completed")
	}
	net, ok := new(big.Int).SetString(e.NetRewardRaw, 10)
	funded, fundedOK := new(big.Int).SetString(c.AmountRaw, 10)
	if !ok || !fundedOK || net.Sign() < 0 || funded.Sign() < 0 || funded.BitLen() > 256 || net.Cmp(funded) > 0 {
		return "", errors.New("allocation_reward_exceeds_funding_or_invalid")
	}
	price, priceErr := decimal.NewFromString(e.RewardUnitUSD)
	capital, capitalErr := decimal.NewFromString(e.CapitalUSDSeconds)
	if priceErr != nil || capitalErr != nil || !price.IsPositive() || !capital.IsPositive() {
		return "", errors.New("allocation_price_or_capital_integral_missing")
	}
	return decimal.NewFromBigInt(net, -18).Mul(price).Mul(decimal.NewFromInt(31536000)).Mul(decimal.NewFromInt(100)).Div(capital).String(), nil
}

// VerifyBOLDClaimProof matches UniversalRewardDistributor.claim exactly:
// keccak256(bytes.concat(keccak256(abi.encode(account,reward,claimable)))),
// sorted-pair OpenZeppelin MerkleProof. It proves cumulative entitlement only,
// NOT the campaign, funding period, vault eligibility or an APR. Pending roots
// must never be substituted for the active root read at the snapshot.
// It does not subtract claimed(account,reward) or establish an unclaimed balance.
func VerifyBOLDClaimProof(root common.Hash, account, reward common.Address, claimable *big.Int, proof []common.Hash) bool {
	if root == (common.Hash{}) || account == (common.Address{}) || reward != boldRouteToken || claimable == nil || claimable.Sign() < 0 || claimable.BitLen() > 256 || len(proof) > 64 {
		return false
	}
	encoded := append(common.LeftPadBytes(account.Bytes(), 32), common.LeftPadBytes(reward.Bytes(), 32)...)
	encoded = append(encoded, common.LeftPadBytes(claimable.Bytes(), 32)...)
	hash := crypto.Keccak256Hash(crypto.Keccak256(encoded))
	for _, sibling := range proof {
		if bytes.Compare(hash[:], sibling[:]) < 0 {
			hash = crypto.Keccak256Hash(hash[:], sibling[:])
		} else {
			hash = crypto.Keccak256Hash(sibling[:], hash[:])
		}
	}
	return hash == root
}

type boldResearchRun struct {
	owner  *BOLDCampaignInspector
	report *BOLDCampaignReport
	next   time.Time
	count  uint64
}

func (s *boldResearchRun) reserve(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.report.RPCRequests >= boldResearchMaxRPC {
		return errors.New("bold_campaign_rpc_budget_exceeded")
	}
	if delay := time.Until(s.next); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	s.next = time.Now().Add(200 * time.Millisecond)
	s.report.RPCRequests++
	return nil
}

func (s *boldResearchRun) header(ctx context.Context, read func(context.Context, *big.Int) (*types.Header, error), n uint64) (*types.Header, error) {
	if err := s.reserve(ctx); err != nil {
		return nil, err
	}
	h, err := read(ctx, new(big.Int).SetUint64(n))
	if err != nil || h == nil || h.Number == nil || !h.Number.IsUint64() || h.Number.Uint64() != n {
		return nil, errors.New("bold_campaign_header_unavailable")
	}
	return h, nil
}

func (s *boldResearchRun) batch(ctx context.Context, reads []snapshotRead) ([][]interface{}, error) {
	a, _ := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	inputs := make([]multicallInput, len(reads))
	parsers := make([]abi.ABI, len(reads))
	for i, read := range reads {
		parsed, err := abi.JSON(strings.NewReader(read.abiJSON))
		if err != nil {
			return nil, errors.New("bold_campaign_invalid_abi")
		}
		parsers[i] = parsed
		data, err := parsed.Pack(read.method, read.args...)
		if err != nil {
			return nil, errors.New("bold_campaign_invalid_call")
		}
		inputs[i] = multicallInput{Target: read.target, AllowFailure: true, CallData: data}
	}
	data, err := a.Pack("aggregate3", inputs)
	if err != nil {
		return nil, errors.New("bold_campaign_invalid_multicall")
	}
	if err := s.reserve(ctx); err != nil {
		return nil, err
	}
	data, err = s.owner.state.CallContract(ctx, ethereum.CallMsg{To: &stakeDAOMulticallAddress, Data: data}, new(big.Int).SetUint64(s.report.BlockNumber))
	if err != nil {
		return nil, errors.New("bold_campaign_contract_state_unavailable")
	}
	out, err := a.Unpack("aggregate3", data)
	if err != nil || len(out) != 1 {
		return nil, errors.New("bold_campaign_invalid_multicall_output")
	}
	items := *abi.ConvertType(out[0], new([]multicallOutput)).(*[]multicallOutput)
	if len(items) != len(reads) {
		return nil, errors.New("bold_campaign_invalid_multicall_output")
	}
	values := make([][]interface{}, len(reads))
	for i, item := range items {
		if !item.Success {
			return nil, errors.New("bold_campaign_getter_unavailable")
		}
		values[i], err = parsers[i].Unpack(reads[i].method, item.ReturnData)
		if err != nil {
			return nil, errors.New("bold_campaign_invalid_getter_output")
		}
	}
	return values, nil
}

func (s *boldResearchRun) readRoute(ctx context.Context) error {
	r := s.report
	controller := common.HexToAddress(PROTOCOL_CONTROLLER_ETH_ARB)
	reads := []snapshotRead{readContract(controller, protocolControllerABI, "gauge", boldRouteGauge),
		readContract(boldRouteReceiver, rewardRouteABI, "rewardVault"), readContract(boldRouteReceiver, rewardRouteABI, "rewardRouter"),
		readContract(boldRouteReceiver, rewardRouteABI, "legacyRewardReceiver"), readContract(boldRouteRouter, rewardRouteABI, "PROTOCOL_CONTROLLER"),
		readContract(boldRouteRouter, rewardRouteABI, "merkleDistributor"), readContract(boldCampaignDistributor, rewardRouteABI, "nbIncentives"),
		readContract(boldCampaignDistributor, boldCampaignResearchABI, "root"), readContract(boldCampaignDistributor, boldCampaignResearchABI, "ipfsHash"),
		readContract(boldRouteToken, erc20DecimalsABI, "decimals")}
	v, err := s.batch(ctx, reads)
	if err != nil {
		return err
	}
	if len(v[0]) != 5 || v[0][0] != r.Vault || v[0][1] != common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e") || v[0][2] != boldRouteReceiver || v[0][3] != [4]byte{0xc7, 0x15, 0xe3, 0x73} || v[0][4] != false {
		return errors.New("bold_campaign_registration_mismatch_or_shutdown")
	}
	for i, expected := range []common.Address{r.Vault, boldRouteRouter, boldRouteLegacy, controller, boldRouteExecutor} {
		if len(v[i+1]) != 1 || v[i+1][0] != expected {
			return errors.New("bold_campaign_reward_route_changed")
		}
	}
	if len(v[6]) != 1 || len(v[7]) != 1 || len(v[8]) != 1 {
		return errors.New("bold_campaign_invalid_metadata")
	}
	n, ok := v[6][0].(*big.Int)
	root, rootOK := v[7][0].([32]byte)
	ipfs, ipfsOK := v[8][0].([32]byte)
	if !ok || n == nil || !n.IsUint64() || !rootOK || !ipfsOK {
		return errors.New("bold_campaign_invalid_metadata")
	}
	if len(v[9]) != 1 || v[9][0] != uint8(18) {
		return errors.New("bold_campaign_reward_decimals_changed")
	}
	s.count, r.Root, r.IPFSHash = n.Uint64(), root, ipfs
	return nil
}

func boldResearchValidateLog(l types.Log, from, through uint64) error {
	if l.Removed || l.BlockNumber < from || l.BlockNumber > through || l.BlockHash == (common.Hash{}) || l.TxHash == (common.Hash{}) || len(l.Topics) == 0 {
		return errors.New("bold_campaign_invalid_log")
	}
	return nil
}

func boldResearchSameLog(a, b types.Log) bool {
	if a.Address != b.Address || a.BlockNumber != b.BlockNumber || a.BlockHash != b.BlockHash || a.TxHash != b.TxHash || a.TxIndex != b.TxIndex || a.Index != b.Index || len(a.Topics) != len(b.Topics) || !bytes.Equal(a.Data, b.Data) {
		return false
	}
	for i := range a.Topics {
		if a.Topics[i] != b.Topics[i] {
			return false
		}
	}
	return true
}

func boldResearchAddress(h common.Hash) (common.Address, bool) {
	return common.BytesToAddress(h[12:]), bytes.Equal(h[:12], make([]byte, 12))
}

func boldResearchCampaignEvent(l types.Log, event abi.Event) (uint64, bool, error) {
	if l.Address != boldCampaignDistributor || len(l.Topics) != 4 || l.Topics[0] != event.ID || len(l.Data) != 160 {
		return 0, false, errors.New("bold_campaign_invalid_event")
	}
	g, gok := boldResearchAddress(l.Topics[2])
	t, tok := boldResearchAddress(l.Topics[3])
	id := new(big.Int).SetBytes(l.Topics[1][:])
	if !gok || !tok || !id.IsUint64() {
		return 0, false, errors.New("bold_campaign_invalid_event")
	}
	if _, err := event.Inputs.NonIndexed().Unpack(l.Data); err != nil {
		return 0, false, errors.New("bold_campaign_invalid_event")
	}
	return id.Uint64(), g == boldRouteGauge && t == boldRouteToken, nil
}

func boldResearchCompareEvent(l types.Log, event abi.Event, v []interface{}) error {
	data, err := event.Inputs.NonIndexed().Pack(v[8], v[2], v[3], v[4], v[7])
	if err != nil || !bytes.Equal(data, l.Data) {
		return errors.New("bold_campaign_event_getter_mismatch")
	}
	return nil
}

func boldResearchVerifyFunding(receipt *types.Receipt, entries map[uint64]types.Log, parsed abi.ABI) error {
	if len(receipt.Logs) > boldResearchMaxEvents {
		return errors.New("bold_campaign_receipt_log_limit")
	}
	pulledEvent, addedEvent := parsed.Events["RewardsPulledToMerkle"], parsed.Events["IncentiveAdded"]
	transfer := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	pulled, funded, intoRouter, intoExecutor, intoDistributor, returned := new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	pulls, campaigns := 0, 0
	seen := map[uint]bool{}
	for _, lp := range receipt.Logs {
		if lp == nil {
			return errors.New("bold_campaign_invalid_receipt_log")
		}
		l := *lp
		if err := boldResearchValidateLog(l, receipt.BlockNumber.Uint64(), receipt.BlockNumber.Uint64()); err != nil {
			return err
		}
		if l.BlockHash != receipt.BlockHash || l.TxHash != receipt.TxHash || l.TxIndex != receipt.TransactionIndex || seen[l.Index] {
			return errors.New("bold_campaign_receipt_identity_mismatch")
		}
		seen[l.Index] = true
		if l.Address == boldRouteRouter && l.Topics[0] == pulledEvent.ID {
			if len(l.Topics) != 3 || len(l.Data) != 32 {
				return errors.New("bold_campaign_invalid_router_event")
			}
			g, gok := boldResearchAddress(l.Topics[1])
			token, tok := boldResearchAddress(l.Topics[2])
			if !gok || !tok {
				return errors.New("bold_campaign_invalid_router_event")
			}
			if token != boldRouteToken {
				continue
			}
			if g != boldRouteGauge {
				return errors.New("bold_campaign_ambiguous_router_pull")
			}
			n := new(big.Int).SetBytes(l.Data)
			if n.Sign() <= 0 || n.BitLen() > 128 {
				return errors.New("bold_campaign_invalid_router_amount")
			}
			pulls++
			pulled.Add(pulled, n)
		}
		if l.Address == boldCampaignDistributor && l.Topics[0] == addedEvent.ID {
			id, match, err := boldResearchCampaignEvent(l, addedEvent)
			if err != nil {
				return err
			}
			if !match {
				if common.BytesToAddress(l.Topics[3][12:]) == boldRouteToken {
					return errors.New("bold_campaign_ambiguous_funding")
				}
				continue
			}
			expected, exists := entries[id]
			if !exists || !boldResearchSameLog(expected, l) {
				return errors.New("bold_campaign_incomplete_or_conflicting_history")
			}
			v, _ := addedEvent.Inputs.NonIndexed().Unpack(l.Data)
			if v[0] != boldRouteExecutor {
				return errors.New("bold_campaign_unreviewed_funding_manager")
			}
			funded.Add(funded, v[4].(*big.Int))
			campaigns++
		}
		if l.Address == boldRouteToken && l.Topics[0] == transfer {
			if len(l.Topics) != 3 || len(l.Data) != 32 {
				return errors.New("bold_campaign_invalid_transfer")
			}
			from, fok := boldResearchAddress(l.Topics[1])
			to, tok := boldResearchAddress(l.Topics[2])
			if !fok || !tok {
				return errors.New("bold_campaign_invalid_transfer")
			}
			n := new(big.Int).SetBytes(l.Data)
			switch {
			case from == boldRouteReceiver && to == boldRouteRouter:
				intoRouter.Add(intoRouter, n)
			case from == boldRouteRouter && to == boldRouteExecutor:
				intoExecutor.Add(intoExecutor, n)
			case from == boldRouteExecutor && to == boldCampaignDistributor:
				intoDistributor.Add(intoDistributor, n)
			case from == boldRouteExecutor && to == boldRouteReceiver:
				returned.Add(returned, n)
			case from == boldRouteExecutor || to == boldRouteExecutor || from == boldRouteRouter || to == boldRouteRouter:
				return errors.New("bold_campaign_unreviewed_transfer_path")
			}
		}
	}
	if pulls != 1 || campaigns == 0 || pulled.Cmp(intoRouter) != 0 || pulled.Cmp(intoExecutor) != 0 || funded.Cmp(intoDistributor) != 0 || new(big.Int).Add(funded, returned).Cmp(pulled) != 0 {
		return errors.New("bold_campaign_funding_not_reconciled")
	}
	return nil
}
