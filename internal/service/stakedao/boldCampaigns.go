package stakedao

// Isolated BOLD/USDC research module. Only explicit research CLI/HTTP calls run
// it; ordinary pool loading and APR/APY calculations never invoke this module.
// Usage: NewBOLDPublishedCampaignInspector(stateRPC, logsRPC).Inspect(ctx, from, block).
// Both clients can be *ethclient.Client; caller owns their lifetime. State calls
// require historical eth_call, log calls need the requested eth_getLogs range.
// No external data API, database, transactions, background jobs or automatic RPC fallback.
//
// Sources: verified CampaignRewardsDistributor + UniversalRewardDistributor at
// https://etherscan.io/address/0xd4898a378ea555595c4e7dbde722b134a3f346d1#code
// and RewardRouter at boldRouteRouter (BOLD_ROUTE_AND_WEETH_MODEL_RU.md).
// A campaign records funding, not the vault's entitlement. The global Merkle
// leaf is (account,reward,cumulativeClaimable), without campaign ID or gauge.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"ABIevm/internal/rpcclient"
	"ABIevm/internal/service/prices"
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
	boldHistoricalReadLimit  = boldResearchMaxRPC - 32 // reserve hash-verification calls
)

// Only local, fixed reason codes have this type. Provider error strings never do.
type boldCampaignFailure string

func (e boldCampaignFailure) Error() string { return string(e) }

var (
	ErrBOLDCampaignInvalidRequest = boldCampaignFailure("bold_campaign_invalid_request")
	ErrBOLDCampaignRPCConfig      = boldCampaignFailure("bold_campaign_rpc_not_configured")
)

func ValidateBOLDCampaignRange(from, block uint64) error {
	if from == 0 || block < from || (block-from)/1500+1 > boldResearchMaxRanges {
		return ErrBOLDCampaignInvalidRequest
	}
	return nil
}

// BOLDCampaignFailureReason is safe to return in HTTP JSON. It does not expose
// RPC URLs, access tokens, arbitrary provider messages or resolver errors.
func BOLDCampaignFailureReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "request_timeout"
	case errors.Is(err, context.Canceled):
		return "request_canceled"
	case errors.Is(err, ErrUnsupportedNetwork):
		return "unsupported_network"
	}
	var failure boldCampaignFailure
	if errors.As(err, &failure) {
		return string(failure)
	}
	return "bold_campaign_check_failed"
}

// InspectBOLDCampaigns is the explicit HTTP/manual entry point. It reads only
// the existing RPC environment settings. Isolated historical prices participate;
// registry loading and ordinary pool/APR aggregation do not. Both transports
// also share the application limiter.
func InspectBOLDCampaigns(ctx context.Context, network string, from, block uint64) (*BOLDCampaignReport, error) {
	if network != "ethereum" {
		return nil, ErrUnsupportedNetwork
	}
	if err := ValidateBOLDCampaignRange(from, block); err != nil {
		return nil, err
	}
	stateURL := strings.TrimSpace(os.Getenv("STAKEDAO_ETHEREUM_RPC_URL"))
	logsURL := strings.TrimSpace(os.Getenv("STAKEDAO_ETHEREUM_LOGS_RPC_URL"))
	if stateURL == "" {
		return nil, ErrBOLDCampaignRPCConfig
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	state, err := rpcclient.DialContext(ctx, stateURL)
	if err != nil {
		return nil, boldCampaignFailure("bold_campaign_state_rpc_unavailable")
	}
	defer state.Close()
	logs := state
	if logsURL != "" && logsURL != stateURL {
		logs, err = rpcclient.DialContext(ctx, logsURL)
		if err != nil {
			return nil, boldCampaignFailure("bold_campaign_logs_rpc_unavailable")
		}
		defer logs.Close()
	}
	report, err := NewBOLDPublishedCampaignInspector(state, logs).Inspect(ctx, from, block)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return report, err
}

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

// BOLDAllocationResolver is a TRUSTED boundary for a reviewed allocation
// model, not an arbitrary price/percentage provider. Implementations must prove
// vault eligibility, campaign attribution, net fees, absence of duplicate
// upstream/vault payouts, and the stated full-period capital model. A sampled
// integral must be identified as an estimate. A Merkle proof
// alone does NOT establish those facts. Nil deliberately leaves APR unknown.
type BOLDAllocationResolver interface {
	ResolveBOLDAllocation(context.Context, BOLDCampaignReport, BOLDCampaignRecord) (BOLDAllocationEvidence, error)
}

// Evidence values are supplied by the reviewed resolver, never inferred from
// campaign funding alone. Finite sampling yields a historical ESTIMATE.
type BOLDAllocationEvidence struct {
	CampaignID         uint64                   `json:"campaignId"`
	SnapshotHash       common.Hash              `json:"snapshotHash" swaggertype:"string"`
	Vault              common.Address           `json:"vault" swaggertype:"string"`
	Start              uint64                   `json:"start"`
	End                uint64                   `json:"end"`
	NetRewardRaw       string                   `json:"netRewardRaw"`
	RewardUnitUSD      string                   `json:"rewardUnitUsd"`
	CapitalUSDSeconds  string                   `json:"capitalUsdSeconds"` // measured or explicitly modeled eligible capital, USD * seconds
	RulesReference     string                   `json:"rulesReference"`
	ValuationReference string                   `json:"valuationReference"`
	EvidenceHash       common.Hash              `json:"evidenceHash" swaggertype:"string"`
	Publication        *BOLDPublicationEvidence `json:"publication,omitempty"`
	Valuation          *BOLDHistoricalValuation `json:"valuation,omitempty"`
}

type BOLDPublicationEvidence struct {
	Status          string      `json:"status"`
	Root            common.Hash `json:"root" swaggertype:"string"`
	Commit          string      `json:"commit"`
	DistributionURL string      `json:"distributionUrl"`
	MerkleURL       string      `json:"merkleUrl"`
	AllocatedAmount string      `json:"allocatedAmount"`
	Recipients      int         `json:"recipients"`
	ProofsVerified  int         `json:"proofsVerified"`
	Scope           string      `json:"scope"`
}

type BOLDHistoricalSample struct {
	RequestedTimestamp  uint64         `json:"requestedTimestamp"`
	BlockNumber         uint64         `json:"blockNumber"`
	BlockTimestamp      uint64         `json:"blockTimestamp"`
	BlockHash           common.Hash    `json:"blockHash" swaggertype:"string"`
	AssetAmount         string         `json:"assetAmount"`
	LPUSD               string         `json:"lpUsd"`
	RewardUSD           string         `json:"rewardUsd"`
	TVLUSD              string         `json:"tvlUsd"`
	LPPriceSource       string         `json:"lpPriceSource"`
	RewardPriceSource   string         `json:"rewardPriceSource"`
	LPPriceContract     common.Address `json:"lpPriceContract" swaggertype:"string"`
	RewardPriceContract common.Address `json:"rewardPriceContract" swaggertype:"string"`
}

type BOLDHistoricalValuation struct {
	Status            string                 `json:"status"`
	Model             string                 `json:"model"`
	CapitalUSDSeconds *string                `json:"capitalUsdSeconds"`
	AverageCapitalUSD *string                `json:"averageCapitalUsd"`
	AverageRewardUSD  *string                `json:"averageRewardUsd"`
	RewardValueUSD    *string                `json:"rewardValueUsd"`
	Samples           []BOLDHistoricalSample `json:"samples"`
	Assumptions       []string               `json:"assumptions"`
}

type BOLDCampaignRecord struct {
	ID               uint64                  `json:"id"`
	Start            uint64                  `json:"start"`
	End              uint64                  `json:"end"`
	AmountRaw        string                  `json:"amountRaw"`
	Amount           string                  `json:"amount"`
	PeriodStatus     string                  `json:"periodStatus"`
	CreationBlock    uint64                  `json:"creationBlock"`
	TransactionHash  common.Hash             `json:"transactionHash" swaggertype:"string"`
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
	BlockHash      common.Hash          `json:"blockHash" swaggertype:"string"`
	FromBlock      uint64               `json:"fromBlock"`
	Coverage       string               `json:"coverage"` // creation blocks, NOT full reward periods
	ScanComplete   bool                 `json:"scanComplete"`
	RPCRequests    int                  `json:"rpcRequests"`
	Vault          common.Address       `json:"vault" swaggertype:"string"`
	Gauge          common.Address       `json:"gauge" swaggertype:"string"`
	Token          common.Address       `json:"token" swaggertype:"string"`
	Distributor    common.Address       `json:"distributor" swaggertype:"string"`
	Root           common.Hash          `json:"root" swaggertype:"string"`
	IPFSHash       common.Hash          `json:"ipfsHash" swaggertype:"string"`
	Campaigns      []BOLDCampaignRecord `json:"campaigns"`
	CurrentAPR     *string              `json:"currentApr"` // historical observations are never current APR
	Reasons        []string             `json:"reasons"`
}

type BOLDCampaignInspector struct {
	state      BOLDCampaignStateReader
	logs       BOLDCampaignLogReader
	allocation BOLDAllocationResolver
	published  bool
}

// NewBOLDPublishedCampaignInspector enables only the reviewed embedded publication
// and historical onchain valuation, never a generic API fallback or a peg price.
func NewBOLDPublishedCampaignInspector(state BOLDCampaignStateReader, logs BOLDCampaignLogReader) *BOLDCampaignInspector {
	return &BOLDCampaignInspector{state: state, logs: logs, published: true}
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
	if a == nil || a.state == nil || a.logs == nil || ValidateBOLDCampaignRange(from, block) != nil {
		return nil, ErrBOLDCampaignInvalidRequest
	}
	r := &BOLDCampaignReport{Status: "unavailable", FromBlock: from, BlockNumber: block,
		Coverage: "campaign_creations_in_requested_blocks_only", Vault: common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad"),
		Gauge: boldRouteGauge, Token: boldRouteToken, Distributor: boldCampaignDistributor, Campaigns: []BOLDCampaignRecord{},
		Reasons: []string{"campaign_registry_not_global", "historical_funding_is_not_current_apr"}}
	s := &boldResearchRun{owner: a, report: r}
	resolver := a.allocation
	var published *boldPublishedResolver
	if a.published && resolver == nil {
		published = &boldPublishedResolver{run: s, headers: map[uint64]*types.Header{}, samples: map[uint64]BOLDHistoricalSample{}, anchors: map[uint64]common.Hash{}}
		resolver = published
	}
	for _, readChain := range []func(context.Context) (*big.Int, error){a.state.ChainID, a.logs.ChainID} {
		if err := s.reserve(ctx); err != nil {
			return nil, err
		}
		chain, err := readChain(ctx)
		if err != nil || chain == nil || chain.Cmp(big.NewInt(1)) != 0 {
			return nil, boldCampaignFailure("bold_campaign_chain_mismatch_or_unavailable")
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
			return nil, boldCampaignFailure("bold_campaign_log_source_mismatch")
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
			return nil, boldCampaignFailure("bold_campaign_log_history_unavailable")
		}
		if len(logs) > boldResearchMaxEvents-len(seen) {
			return nil, boldCampaignFailure("bold_campaign_event_limit")
		}
		for _, l := range logs {
			if err := boldResearchValidateLog(l, start, through); err != nil {
				return nil, err
			}
			key := l.TxHash.Hex() + ":" + new(big.Int).SetUint64(uint64(l.Index)).String()
			if old, exists := seen[key]; exists {
				if !boldResearchSameLog(old, l) {
					return nil, boldCampaignFailure("bold_campaign_conflicting_logs")
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
					return nil, boldCampaignFailure("bold_campaign_id_outside_snapshot_registry")
				}
				if _, duplicate := entries[id]; duplicate {
					return nil, boldCampaignFailure("bold_campaign_duplicate_id")
				}
				entries[id] = l
				if len(entries) > boldResearchMaxCampaigns {
					return nil, boldCampaignFailure("bold_campaign_count_limit")
				}
			} else if l.Address != boldRouteRouter || l.Topics[0] != pulled.ID {
				return nil, boldCampaignFailure("bold_campaign_unexpected_event")
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
				return nil, boldCampaignFailure("bold_campaign_invalid_getter")
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
				return nil, boldCampaignFailure("bold_campaign_receipt_limit")
			}
			if err := s.reserve(ctx); err != nil {
				return nil, err
			}
			receipt, err = a.state.TransactionReceipt(ctx, c.TransactionHash)
			if err != nil || receipt == nil || receipt.Status != types.ReceiptStatusSuccessful || receipt.TxHash != c.TransactionHash || receipt.BlockNumber == nil || !receipt.BlockNumber.IsUint64() || receipt.BlockNumber.Uint64() != c.CreationBlock {
				return nil, boldCampaignFailure("bold_campaign_receipt_unavailable_or_invalid")
			}
			h, err := s.header(ctx, a.state.HeaderByNumber, c.CreationBlock)
			if err != nil || h.Hash() != receipt.BlockHash {
				return nil, boldCampaignFailure("bold_campaign_receipt_block_mismatch")
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
				return nil, boldCampaignFailure("bold_campaign_snapshot_changed")
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
		if resolver == nil {
			continue
		}
		// Pass no caller-mutable report slices to the trusted resolver.
		view := *r
		view.Campaigns = nil
		view.Reasons = append([]string(nil), r.Reasons...)
		evidence, err := resolver.ResolveBOLDAllocation(ctx, view, *c)
		if published != nil && evidence.Publication != nil {
			c.Allocation = &evidence
			c.AllocationStatus = "published_allocation_verified"
		}
		if err != nil {
			c.Reasons = []string{"allocation_evidence_unavailable"}
			var safe boldCampaignFailure
			if errors.As(err, &safe) {
				c.Reasons = []string{string(safe)}
			}
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
		if c.Allocation != nil && c.Allocation.Publication != nil {
			r.Status = "funding_and_allocation_verified_valuation_unavailable"
		}
	}
	for _, c := range r.Campaigns {
		if c.HistoricalAPR != nil {
			r.Status = "funding_verified_with_historical_estimates"
			break
		}
	}
	if published != nil {
		if err := published.verifyAnchors(ctx); err != nil {
			return nil, err
		}
	}
	if resolver != nil {
		checked, err := s.header(ctx, a.state.HeaderByNumber, block)
		if err != nil || checked.Hash() != r.BlockHash {
			return nil, boldCampaignFailure("bold_campaign_snapshot_changed")
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
		return "", boldCampaignFailure("allocation_evidence_unverified_or_mismatched")
	}
	if e.End <= e.Start || e.End > r.BlockTimestamp {
		return "", boldCampaignFailure("allocation_period_not_completed")
	}
	net, ok := new(big.Int).SetString(e.NetRewardRaw, 10)
	funded, fundedOK := new(big.Int).SetString(c.AmountRaw, 10)
	if !ok || !fundedOK || net.Sign() < 0 || funded.Sign() < 0 || funded.BitLen() > 256 || net.Cmp(funded) > 0 {
		return "", boldCampaignFailure("allocation_reward_exceeds_funding_or_invalid")
	}
	price, priceErr := decimal.NewFromString(e.RewardUnitUSD)
	capital, capitalErr := decimal.NewFromString(e.CapitalUSDSeconds)
	if priceErr != nil || capitalErr != nil || !price.IsPositive() || !capital.IsPositive() {
		return "", boldCampaignFailure("allocation_price_or_capital_integral_missing")
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
		return boldCampaignFailure("bold_campaign_rpc_budget_exceeded")
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
		return nil, boldCampaignFailure("bold_campaign_header_unavailable")
	}
	return h, nil
}

func (s *boldResearchRun) batch(ctx context.Context, reads []snapshotRead) ([][]interface{}, error) {
	return s.batchAt(ctx, s.report.BlockNumber, reads)
}

func (s *boldResearchRun) batchAt(ctx context.Context, block uint64, reads []snapshotRead) ([][]interface{}, error) {
	a, _ := abi.JSON(strings.NewReader(stakeDAOMulticallABI))
	inputs := make([]multicallInput, len(reads))
	parsers := make([]abi.ABI, len(reads))
	for i, read := range reads {
		parsed, err := abi.JSON(strings.NewReader(read.abiJSON))
		if err != nil {
			return nil, boldCampaignFailure("bold_campaign_invalid_abi")
		}
		parsers[i] = parsed
		data, err := parsed.Pack(read.method, read.args...)
		if err != nil {
			return nil, boldCampaignFailure("bold_campaign_invalid_call")
		}
		inputs[i] = multicallInput{Target: read.target, AllowFailure: true, CallData: data}
	}
	data, err := a.Pack("aggregate3", inputs)
	if err != nil {
		return nil, boldCampaignFailure("bold_campaign_invalid_multicall")
	}
	if err := s.reserve(ctx); err != nil {
		return nil, err
	}
	data, err = s.owner.state.CallContract(ctx, ethereum.CallMsg{To: &stakeDAOMulticallAddress, Data: data}, new(big.Int).SetUint64(block))
	if err != nil {
		return nil, boldCampaignFailure("bold_campaign_contract_state_unavailable")
	}
	out, err := a.Unpack("aggregate3", data)
	if err != nil || len(out) != 1 {
		return nil, boldCampaignFailure("bold_campaign_invalid_multicall_output")
	}
	items := *abi.ConvertType(out[0], new([]multicallOutput)).(*[]multicallOutput)
	if len(items) != len(reads) {
		return nil, boldCampaignFailure("bold_campaign_invalid_multicall_output")
	}
	values := make([][]interface{}, len(reads))
	for i, item := range items {
		if !item.Success {
			return nil, boldCampaignFailure("bold_campaign_getter_unavailable")
		}
		values[i], err = parsers[i].Unpack(reads[i].method, item.ReturnData)
		if err != nil {
			return nil, boldCampaignFailure("bold_campaign_invalid_getter_output")
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
		return boldCampaignFailure("bold_campaign_registration_mismatch_or_shutdown")
	}
	for i, expected := range []common.Address{r.Vault, boldRouteRouter, boldRouteLegacy, controller, boldRouteExecutor} {
		if len(v[i+1]) != 1 || v[i+1][0] != expected {
			return boldCampaignFailure("bold_campaign_reward_route_changed")
		}
	}
	if len(v[6]) != 1 || len(v[7]) != 1 || len(v[8]) != 1 {
		return boldCampaignFailure("bold_campaign_invalid_metadata")
	}
	n, ok := v[6][0].(*big.Int)
	root, rootOK := v[7][0].([32]byte)
	ipfs, ipfsOK := v[8][0].([32]byte)
	if !ok || n == nil || !n.IsUint64() || !rootOK || !ipfsOK {
		return boldCampaignFailure("bold_campaign_invalid_metadata")
	}
	if len(v[9]) != 1 || v[9][0] != uint8(18) {
		return boldCampaignFailure("bold_campaign_reward_decimals_changed")
	}
	s.count, r.Root, r.IPFSHash = n.Uint64(), root, ipfs
	return nil
}

func boldResearchValidateLog(l types.Log, from, through uint64) error {
	if l.Removed || l.BlockNumber < from || l.BlockNumber > through || l.BlockHash == (common.Hash{}) || l.TxHash == (common.Hash{}) || len(l.Topics) == 0 {
		return boldCampaignFailure("bold_campaign_invalid_log")
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
		return 0, false, boldCampaignFailure("bold_campaign_invalid_event")
	}
	g, gok := boldResearchAddress(l.Topics[2])
	t, tok := boldResearchAddress(l.Topics[3])
	id := new(big.Int).SetBytes(l.Topics[1][:])
	if !gok || !tok || !id.IsUint64() {
		return 0, false, boldCampaignFailure("bold_campaign_invalid_event")
	}
	if _, err := event.Inputs.NonIndexed().Unpack(l.Data); err != nil {
		return 0, false, boldCampaignFailure("bold_campaign_invalid_event")
	}
	return id.Uint64(), g == boldRouteGauge && t == boldRouteToken, nil
}

func boldResearchCompareEvent(l types.Log, event abi.Event, v []interface{}) error {
	data, err := event.Inputs.NonIndexed().Pack(v[8], v[2], v[3], v[4], v[7])
	if err != nil || !bytes.Equal(data, l.Data) {
		return boldCampaignFailure("bold_campaign_event_getter_mismatch")
	}
	return nil
}

func boldResearchVerifyFunding(receipt *types.Receipt, entries map[uint64]types.Log, parsed abi.ABI) error {
	if len(receipt.Logs) > boldResearchMaxEvents {
		return boldCampaignFailure("bold_campaign_receipt_log_limit")
	}
	pulledEvent, addedEvent := parsed.Events["RewardsPulledToMerkle"], parsed.Events["IncentiveAdded"]
	transfer := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	pulled, funded, intoRouter, intoExecutor, intoDistributor, returned := new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	pulls, campaigns := 0, 0
	seen := map[uint]bool{}
	for _, lp := range receipt.Logs {
		if lp == nil {
			return boldCampaignFailure("bold_campaign_invalid_receipt_log")
		}
		l := *lp
		if err := boldResearchValidateLog(l, receipt.BlockNumber.Uint64(), receipt.BlockNumber.Uint64()); err != nil {
			return err
		}
		if l.BlockHash != receipt.BlockHash || l.TxHash != receipt.TxHash || l.TxIndex != receipt.TransactionIndex || seen[l.Index] {
			return boldCampaignFailure("bold_campaign_receipt_identity_mismatch")
		}
		seen[l.Index] = true
		if l.Address == boldRouteRouter && l.Topics[0] == pulledEvent.ID {
			if len(l.Topics) != 3 || len(l.Data) != 32 {
				return boldCampaignFailure("bold_campaign_invalid_router_event")
			}
			g, gok := boldResearchAddress(l.Topics[1])
			token, tok := boldResearchAddress(l.Topics[2])
			if !gok || !tok {
				return boldCampaignFailure("bold_campaign_invalid_router_event")
			}
			if token != boldRouteToken {
				continue
			}
			if g != boldRouteGauge {
				return boldCampaignFailure("bold_campaign_ambiguous_router_pull")
			}
			n := new(big.Int).SetBytes(l.Data)
			if n.Sign() <= 0 || n.BitLen() > 128 {
				return boldCampaignFailure("bold_campaign_invalid_router_amount")
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
					return boldCampaignFailure("bold_campaign_ambiguous_funding")
				}
				continue
			}
			expected, exists := entries[id]
			if !exists || !boldResearchSameLog(expected, l) {
				return boldCampaignFailure("bold_campaign_incomplete_or_conflicting_history")
			}
			v, _ := addedEvent.Inputs.NonIndexed().Unpack(l.Data)
			if v[0] != boldRouteExecutor {
				return boldCampaignFailure("bold_campaign_unreviewed_funding_manager")
			}
			funded.Add(funded, v[4].(*big.Int))
			campaigns++
		}
		if l.Address == boldRouteToken && l.Topics[0] == transfer {
			if len(l.Topics) != 3 || len(l.Data) != 32 {
				return boldCampaignFailure("bold_campaign_invalid_transfer")
			}
			from, fok := boldResearchAddress(l.Topics[1])
			to, tok := boldResearchAddress(l.Topics[2])
			if !fok || !tok {
				return boldCampaignFailure("bold_campaign_invalid_transfer")
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
				return boldCampaignFailure("bold_campaign_unreviewed_transfer_path")
			}
		}
	}
	if pulls != 1 || campaigns == 0 || pulled.Cmp(intoRouter) != 0 || pulled.Cmp(intoExecutor) != 0 || funded.Cmp(intoDistributor) != 0 || new(big.Int).Add(funded, returned).Cmp(pulled) != 0 {
		return boldCampaignFailure("bold_campaign_funding_not_reconciled")
	}
	return nil
}

// The embedded publication is a reviewed, root-specific data route, not an API.
// Baseline attribution was reconciled against pinned full trees offline (see
// examples/stakedao-bold-published-root-verification.json). At runtime we bind
// it to the live root and verify every beneficiary's target-root proof again.
// Published eligibility weights are NOT an independently replayed share history.
type boldReviewedPublication struct {
	Root      common.Hash                          `json:"root"`
	Block     uint64                               `json:"block"`
	Timestamp uint64                               `json:"timestamp"`
	Commit    string                               `json:"commit"`
	Sources   []struct{ Path, URL, SHA256 string } `json:"sources"`
	Campaigns []struct {
		ID               uint64 `json:"id"`
		Start, End       uint64
		FundedRaw        string                                   `json:"fundedRaw"`
		AllocatedRaw     string                                   `json:"allocatedRaw"`
		RateRawPerSecond string                                   `json:"rateRawPerSecond"`
		Users            []struct{ User, Balance, Amount string } `json:"users"`
	} `json:"campaigns"`
	Proofs []struct {
		Account string        `json:"account"`
		Delta   string        `json:"campaignsAllocatedRaw"`
		Before  string        `json:"cumulativeBeforeRaw"`
		After   string        `json:"cumulativeAfterRaw"`
		Target  string        `json:"cumulativeAtTargetRootRaw"`
		Proof   []common.Hash `json:"targetRootProof"`
	} `json:"proofs"`
}

var boldReviewedCache struct {
	sync.Once
	data *boldReviewedPublication
	err  error
}

func reviewedBOLDPublication() (*boldReviewedPublication, error) {
	boldReviewedCache.Do(func() {
		boldReviewedCache.data, boldReviewedCache.err = decodeBOLDPublication([]byte(boldReviewedPublicationJSON))
	})
	return boldReviewedCache.data, boldReviewedCache.err
}

func boldPublicationUint(raw string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(raw, 10)
	if !ok || n.Sign() < 0 || n.BitLen() > 256 {
		return nil, boldCampaignFailure("published_distribution_invalid_integer")
	}
	return n, nil
}

func decodeBOLDPublication(raw []byte) (*boldReviewedPublication, error) {
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != boldReviewedPublicationSHA256 {
		return nil, boldCampaignFailure("published_distribution_hash_mismatch")
	}
	var p boldReviewedPublication
	if json.Unmarshal(raw, &p) != nil || p.Root == (common.Hash{}) || len(p.Campaigns) != 2 || len(p.Proofs) != 15 {
		return nil, boldCampaignFailure("published_distribution_invalid")
	}
	totals := map[common.Address]*big.Int{}
	ids := map[uint64]bool{}
	for _, c := range p.Campaigns {
		if ids[c.ID] || c.End <= c.Start || len(c.Users) == 0 {
			return nil, boldCampaignFailure("published_distribution_invalid")
		}
		ids[c.ID] = true
		funded, err := boldPublicationUint(c.FundedRaw)
		if err != nil {
			return nil, err
		}
		allocated, err := boldPublicationUint(c.AllocatedRaw)
		if err != nil {
			return nil, err
		}
		rate, err := boldPublicationUint(c.RateRawPerSecond)
		if err != nil {
			return nil, err
		}
		duration := new(big.Int).SetUint64(c.End - c.Start)
		if new(big.Int).Div(new(big.Int).Set(funded), duration).Cmp(rate) != 0 || new(big.Int).Mul(rate, duration).Cmp(allocated) != 0 {
			return nil, boldCampaignFailure("published_distribution_rate_mismatch")
		}
		weights := new(big.Int)
		seen := map[common.Address]bool{}
		for _, u := range c.Users {
			w, err := boldPublicationUint(u.Balance)
			if err != nil || w.Sign() <= 0 {
				return nil, boldCampaignFailure("published_distribution_weight_invalid")
			}
			weights.Add(weights, w)
			a := common.HexToAddress(u.User)
			if !common.IsHexAddress(u.User) || a == (common.Address{}) || seen[a] {
				return nil, boldCampaignFailure("published_distribution_duplicate_recipient")
			}
			seen[a] = true
		}
		used := new(big.Int)
		for i, u := range c.Users {
			w, _ := boldPublicationUint(u.Balance)
			amount, err := boldPublicationUint(u.Amount)
			if err != nil {
				return nil, err
			}
			expected := new(big.Int).Div(new(big.Int).Mul(allocated, w), weights)
			if i == len(c.Users)-1 {
				expected.Sub(allocated, used)
			}
			if amount.Cmp(expected) != 0 {
				return nil, boldCampaignFailure("published_distribution_payout_mismatch")
			}
			used.Add(used, amount)
			a := common.HexToAddress(u.User)
			if totals[a] == nil {
				totals[a] = new(big.Int)
			}
			totals[a].Add(totals[a], amount)
		}
		if used.Cmp(allocated) != 0 {
			return nil, boldCampaignFailure("published_distribution_sum_mismatch")
		}
	}
	seen := map[common.Address]bool{}
	for _, v := range p.Proofs {
		a := common.HexToAddress(v.Account)
		target, err := boldPublicationUint(v.Target)
		if err != nil {
			return nil, err
		}
		delta, err := boldPublicationUint(v.Delta)
		if err != nil {
			return nil, err
		}
		before, err := boldPublicationUint(v.Before)
		if err != nil {
			return nil, err
		}
		after, err := boldPublicationUint(v.After)
		if err != nil {
			return nil, err
		}
		if seen[a] || totals[a] == nil || totals[a].Cmp(delta) != 0 || new(big.Int).Add(before, delta).Cmp(after) != 0 || after.Cmp(target) != 0 || !VerifyBOLDClaimProof(p.Root, a, boldRouteToken, target, v.Proof) {
			return nil, boldCampaignFailure("published_distribution_proof_or_delta_mismatch")
		}
		seen[a] = true
	}
	if len(totals) != len(seen) {
		return nil, boldCampaignFailure("published_distribution_missing_proof")
	}
	return &p, nil
}

type boldPublishedResolver struct {
	run     *boldResearchRun
	headers map[uint64]*types.Header
	samples map[uint64]BOLDHistoricalSample
	anchors map[uint64]common.Hash
	// Optional internal test seam; production always reads historical contracts.
	sampleOverride func(context.Context, uint64) (BOLDHistoricalSample, error)
}

func (v *boldPublishedResolver) ResolveBOLDAllocation(ctx context.Context, r BOLDCampaignReport, c BOLDCampaignRecord) (BOLDAllocationEvidence, error) {
	e := BOLDAllocationEvidence{CampaignID: c.ID, SnapshotHash: r.BlockHash, Vault: r.Vault, Start: c.Start, End: c.End}
	p, err := reviewedBOLDPublication()
	if err != nil {
		return e, err
	}
	if r.Root != p.Root {
		return e, boldCampaignFailure("publication_not_registered_for_root")
	}
	if r.BlockNumber < p.Block || r.BlockTimestamp < p.Timestamp || r.Gauge != boldRouteGauge || r.Token != boldRouteToken || r.Vault != common.HexToAddress("0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad") || c.FundingStatus != "verified_router_campaign_funding" {
		return e, boldCampaignFailure("publication_snapshot_or_vault_mismatch")
	}
	found := false
	for _, item := range p.Campaigns {
		if item.ID != c.ID {
			continue
		}
		found = true
		if item.Start != c.Start || item.End != c.End || item.FundedRaw != c.AmountRaw {
			return e, boldCampaignFailure("publication_campaign_mismatch")
		}
		e.NetRewardRaw = item.AllocatedRaw
		e.RulesReference = "stake-dao/merkl-toolkit@" + p.Commit + ":reviewed_period_weights_and_cumulative_delta"
		e.EvidenceHash = common.HexToHash("0x" + boldReviewedPublicationSHA256)
		e.Publication = &BOLDPublicationEvidence{Status: "verified_live_root_and_recipient_proofs", Root: p.Root, Commit: p.Commit,
			AllocatedAmount: decimal.NewFromBigInt(func() *big.Int { n, _ := boldPublicationUint(item.AllocatedRaw); return n }(), -18).String(), Recipients: len(item.Users), ProofsVerified: len(item.Users), Scope: "vault_beneficiary_earned_entitlements_not_claimed_transfers"}
		for _, s := range p.Sources {
			if s.Path == "data/distributions/1795111271/distribution.json" {
				e.Publication.DistributionURL = s.URL
			}
			if s.Path == "data/distributions/1795176071/merkle.json" {
				e.Publication.MerkleURL = s.URL
			}
		}
	}
	if !found {
		return e, boldCampaignFailure("campaign_not_in_registered_publication")
	}
	h := &BOLDHistoricalValuation{Status: "unavailable", Model: "three_point_trapezoidal_historical_tvl", Samples: []BOLDHistoricalSample{}, Assumptions: []string{
		"historical_state_at_or_before_each_sample_timestamp", "linear_interpolation_between_three_samples", "uniform_earned_reward_rate_within_campaign", "published_eligibility_weights_not_independently_replayed", "no_upstream_or_vault_reward_double_counting", "historical_estimate_not_current_apr_not_apy"}}
	e.Valuation = h
	if c.End <= c.Start || c.End > r.BlockTimestamp {
		return e, boldCampaignFailure("allocation_period_not_completed")
	}
	for _, t := range []uint64{c.Start, c.Start + (c.End-c.Start)/2, c.End} {
		point, err := v.sample(ctx, t)
		if err != nil {
			return e, err
		}
		h.Samples = append(h.Samples, point)
	}
	capital, price, err := integrateBOLDHistory(h.Samples, c.Start, c.End)
	if err != nil {
		return e, err
	}
	e.CapitalUSDSeconds, e.RewardUnitUSD = capital.String(), price.String()
	e.ValuationReference = "historical_curve_lp_nav_and_bold_ema_usd_trapezoidal_three_point"
	average := capital.Div(decimal.NewFromInt(int64(c.End - c.Start))).String()
	tokens, _ := boldPublicationUint(e.NetRewardRaw)
	reward := decimal.NewFromBigInt(tokens, -18).Mul(price).String()
	h.Status = "estimate_available"
	h.CapitalUSDSeconds = &e.CapitalUSDSeconds
	h.AverageRewardUSD = &e.RewardUnitUSD
	h.AverageCapitalUSD = &average
	h.RewardValueUSD = &reward
	return e, nil
}

func integrateBOLDHistory(points []BOLDHistoricalSample, start, end uint64) (decimal.Decimal, decimal.Decimal, error) {
	fail := func() (decimal.Decimal, decimal.Decimal, error) {
		return decimal.Zero, decimal.Zero, boldCampaignFailure("historical_samples_invalid")
	}
	if len(points) != 3 || end <= start || points[0].RequestedTimestamp != start || points[2].RequestedTimestamp != end {
		return fail()
	}
	capital, rewardPrice := decimal.Zero, decimal.Zero
	for i, p := range points {
		tvl, err := decimal.NewFromString(p.TVLUSD)
		if err != nil || tvl.IsNegative() {
			return fail()
		}
		price, err := decimal.NewFromString(p.RewardUSD)
		if err != nil || !price.IsPositive() {
			return fail()
		}
		if i == 0 {
			continue
		}
		q := points[i-1]
		if p.RequestedTimestamp <= q.RequestedTimestamp {
			return fail()
		}
		previousTVL, err := decimal.NewFromString(q.TVLUSD)
		if err != nil {
			return fail()
		}
		previousPrice, err := decimal.NewFromString(q.RewardUSD)
		if err != nil {
			return fail()
		}
		seconds := decimal.NewFromBigInt(new(big.Int).SetUint64(p.RequestedTimestamp-q.RequestedTimestamp), 0)
		capital = capital.Add(previousTVL.Add(tvl).Mul(seconds).Div(decimal.NewFromInt(2)))
		rewardPrice = rewardPrice.Add(previousPrice.Add(price).Mul(seconds).Div(decimal.NewFromInt(2)))
	}
	if !capital.IsPositive() {
		return decimal.Zero, decimal.Zero, boldCampaignFailure("historical_capital_is_zero")
	}
	return capital, rewardPrice.Div(decimal.NewFromBigInt(new(big.Int).SetUint64(end-start), 0)), nil
}

func (v *boldPublishedResolver) readHeader(ctx context.Context, n uint64) (*types.Header, error) {
	if h := v.headers[n]; h != nil {
		return h, nil
	}
	if v.run.report.RPCRequests >= boldHistoricalReadLimit {
		return nil, boldCampaignFailure("bold_campaign_rpc_budget_exceeded")
	}
	h, err := v.run.header(ctx, v.run.owner.state.HeaderByNumber, n)
	if err != nil {
		return nil, err
	}
	if h.Time > v.run.report.BlockTimestamp {
		return nil, boldCampaignFailure("historical_header_time_invalid")
	}
	v.headers[n] = h
	return h, nil
}

func (v *boldPublishedResolver) boundary(ctx context.Context, t uint64) (*types.Header, error) {
	r := v.run.report
	if t == r.BlockTimestamp {
		h, err := v.readHeader(ctx, r.BlockNumber)
		if err != nil {
			return nil, err
		}
		if h.Time != t || h.Hash() != r.BlockHash {
			return nil, boldCampaignFailure("historical_sample_reorganized")
		}
		v.anchors[r.BlockNumber] = h.Hash()
		return h, nil
	}
	if t > r.BlockTimestamp {
		return nil, boldCampaignFailure("historical_timestamp_outside_snapshot")
	}
	span := (r.BlockTimestamp - t) / 12
	guess := uint64(0)
	if r.BlockNumber > span {
		guess = r.BlockNumber - span
	}
	low := uint64(0)
	if guess > 7200 {
		low = guess - 7200
	}
	high := r.BlockNumber
	if r.BlockNumber-guess > 7200 {
		high = guess + 7200
	}
	lh, err := v.readHeader(ctx, low)
	if err != nil {
		return nil, err
	}
	hh, err := v.readHeader(ctx, high)
	if err != nil {
		return nil, err
	}
	if lh.Time > t || hh.Time <= t {
		low, high = 0, r.BlockNumber
		lh, err = v.readHeader(ctx, 0)
		if err != nil {
			return nil, err
		}
		hh, err = v.readHeader(ctx, high)
		if err != nil {
			return nil, err
		}
	}
	if lh.Time > t || hh.Time <= t {
		return nil, boldCampaignFailure("historical_boundary_unavailable")
	}
	for high-low > 1 {
		mid := low + (high-low)/2
		h, err := v.readHeader(ctx, mid)
		if err != nil {
			return nil, err
		}
		if h.Time <= t {
			low, lh = mid, h
		} else {
			high, hh = mid, h
		}
	}
	if lh.Time > t || t-lh.Time > 120 || hh.Time <= t {
		return nil, boldCampaignFailure("historical_boundary_gap")
	}
	v.anchors[low], v.anchors[high] = lh.Hash(), hh.Hash()
	return lh, nil
}

type boldHistoricalCaller struct {
	run   *boldResearchRun
	block uint64
}

func (c boldHistoricalCaller) CallContract(ctx context.Context, msg ethereum.CallMsg, b *big.Int) ([]byte, error) {
	if b == nil || !b.IsUint64() || b.Uint64() != c.block {
		return nil, boldCampaignFailure("historical_price_block_mismatch")
	}
	if c.run.report.RPCRequests >= boldHistoricalReadLimit {
		return nil, boldCampaignFailure("bold_campaign_rpc_budget_exceeded")
	}
	if err := c.run.reserve(ctx); err != nil {
		return nil, err
	}
	return c.run.owner.state.CallContract(ctx, msg, b)
}

const boldHistoricalVaultABI = `[{"type":"function","name":"asset","inputs":[],"outputs":[{"type":"address"}]}]`

func (v *boldPublishedResolver) sample(ctx context.Context, t uint64) (BOLDHistoricalSample, error) {
	if v.sampleOverride != nil {
		return v.sampleOverride(ctx, t)
	}
	if p, ok := v.samples[t]; ok {
		return p, nil
	}
	p := BOLDHistoricalSample{RequestedTimestamp: t}
	h, err := v.boundary(ctx, t)
	if err != nil {
		return p, err
	}
	p.BlockNumber, p.BlockTimestamp, p.BlockHash = h.Number.Uint64(), h.Time, h.Hash()
	if v.run.report.RPCRequests >= boldHistoricalReadLimit {
		return p, boldCampaignFailure("bold_campaign_rpc_budget_exceeded")
	}
	lp := common.HexToAddress("0xefc6516323fbd28e80b85a497b65a86243a54b3e")
	values, err := v.run.batchAt(ctx, p.BlockNumber, []snapshotRead{readContract(v.run.report.Vault, boldHistoricalVaultABI, "asset"), readContract(v.run.report.Vault, rewardVaultERC4626ABI, "totalAssets"), readContract(lp, erc20DecimalsABI, "decimals")})
	if err != nil {
		return p, boldCampaignFailure("historical_vault_state_unavailable")
	}
	if len(values[0]) != 1 || values[0][0] != lp || len(values[1]) != 1 || len(values[2]) != 1 || values[2][0] != uint8(18) {
		return p, boldCampaignFailure("historical_asset_identity_mismatch")
	}
	n, ok := values[1][0].(*big.Int)
	if !ok || n == nil || n.Sign() < 0 {
		return p, boldCampaignFailure("historical_assets_invalid")
	}
	assets := decimal.NewFromBigInt(n, -18)
	p.AssetAmount = assets.String()
	quotes := prices.NewSnapshotPriceService(h.Time)
	if err := quotes.EnsurePricesAt(ctx, 1, []common.Address{boldRouteToken, lp}, prices.Snapshot{Caller: boldHistoricalCaller{v.run, p.BlockNumber}, Block: p.BlockNumber, Timestamp: h.Time}); err != nil {
		if v.run.report.RPCRequests >= boldHistoricalReadLimit {
			return p, boldCampaignFailure("bold_campaign_rpc_budget_exceeded")
		}
		return p, boldCampaignFailure("historical_price_unavailable")
	}
	bq, be := quotes.GetTokenQuote(ctx, boldRouteToken, 1)
	lq, le := quotes.GetTokenQuote(ctx, lp, 1)
	if be != nil || le != nil || bq.BlockNumber != p.BlockNumber || lq.BlockNumber != p.BlockNumber {
		return p, boldCampaignFailure("historical_price_block_mismatch")
	}
	p.LPUSD, p.RewardUSD, p.TVLUSD = lq.USD.String(), bq.USD.String(), assets.Mul(lq.USD).String()
	p.LPPriceSource, p.RewardPriceSource = lq.Source, bq.Source
	p.LPPriceContract, p.RewardPriceContract = lq.SourceContract, bq.SourceContract
	checked, err := v.run.header(ctx, v.run.owner.state.HeaderByNumber, p.BlockNumber)
	if err != nil || checked.Hash() != p.BlockHash {
		return p, boldCampaignFailure("historical_sample_reorganized")
	}
	v.samples[t] = p
	return p, nil
}

func (v *boldPublishedResolver) verifyAnchors(ctx context.Context) error {
	numbers := make([]uint64, 0, len(v.anchors))
	for n := range v.anchors {
		numbers = append(numbers, n)
	}
	sort.Slice(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })
	// Never hide exhausted history/price budget as a failed overall funding scan.
	// Successfully sampled historical blocks still must be verified before use.
	for _, n := range numbers {
		h, err := v.run.header(ctx, v.run.owner.state.HeaderByNumber, n)
		if err != nil {
			return err
		}
		if h.Hash() != v.anchors[n] {
			return boldCampaignFailure("historical_sample_reorganized")
		}
	}
	return nil
}

const boldReviewedPublicationSHA256 = "ca06466c106c1c845e14794e863033146677729ad8ca5661a048e028757a34e8"

// Root-specific reviewed data; provenance and full source hashes are part of the bundle.
const boldReviewedPublicationJSON = `{"root":"0xbeafd406e6d206e131a4a37f4fa5fdc6e895422dcb1ff391bc8a54ff6c2d0af8","block":26456172,"timestamp":1795176071,"commit":"84f2afdc9c2ac8d8f665c5219d0da348aa26e444","sources":[{"path":"data/distributions/1795176071/merkle.json","url":"https://raw.githubusercontent.com/stake-dao/merkl-toolkit/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795176071/merkle.json","sha256":"ef0e0e2180465aefd5ba3e2be44feb59424d063732a8e5f2437a4f81c128f755"},{"path":"data/distributions/1795176071/distribution.json","url":"https://raw.githubusercontent.com/stake-dao/merkl-toolkit/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795176071/distribution.json","sha256":"322b54668be80c5475ec74566fb60fe27154faf4c424cf2d3aa3807785f24f6c"},{"path":"data/distributions/1795111271/distribution.json","url":"https://raw.githubusercontent.com/stake-dao/merkl-toolkit/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795111271/distribution.json","sha256":"d1c5497a4d04af943f81057bfbe1dd51ddc15fbe214f93d656a3469aa33afe77"},{"path":"data/distributions/1795111271/gauges/0x543D3Ee7Cfa494E8ffCC18364D130C8eED4B90ad.json","url":"https://raw.githubusercontent.com/stake-dao/merkl-toolkit/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795111271/gauges/0x543D3Ee7Cfa494E8ffCC18364D130C8eED4B90ad.json","sha256":"b7b42db4dbe8cc7dfdfb3394d14a6720dce22a4b5a6a3e5bfc85bdf249986051"},{"path":"data/distributions/1795089695/merkle.json","url":"https://raw.githubusercontent.com/stake-dao/merkl-toolkit/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795089695/merkle.json","sha256":"37585aed8224c09037df19892c19315a65d900b3ed36a7ba21d024a45d67e81b"},{"path":"data/distributions/1795111271/merkle.json","url":"https://raw.githubusercontent.com/stake-dao/merkl-toolkit/84f2afdc9c2ac8d8f665c5219d0da348aa26e444/data/distributions/1795111271/merkle.json","sha256":"63166c248da2d8b1a7b4c77097e40aa5a69cf1318234e6a1d160ef3d2b02929f"}],"proofs":[{"account":"0x01e246880e3d59f1af2f404a456026af5577b4a8","campaignsAllocatedRaw":"14471564992127017","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"14471564992127017","cumulativeAtTargetRootRaw":"14471564992127017","targetRootProof":["0x4021aa4c9548fe73e0cf99a99296114ec575be5c8c215f737af69fa6e01ef10a","0xea6080c52c3504b061f52428450ccee29008f176fd85ae72cf302a9d23089db6","0xdbc7ffaccac2eb5eb10ee8cdc653e075f300e2c947f38f9cc4f8f1bf590b9c9f","0x396920a0b71ce159e405e82123bafa9fc4ff8c10fd1ed29c97985c0e6f262fe8","0x62f46ecccd8e3bcde7a48481a6f1decd5ba0cfb9121ec4c6d9073e35032267b4","0x5e28347137c9ae7dab7eec2557a012f0555a234b39ab19f2bfa5c49326e401f7","0xfe17e3ac2f8a782948d8d3da8be4d80f66a14567b31128a88bff069887241ade","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x07d398f9cbf0f07db51ae4c4105f38824821aa48","campaignsAllocatedRaw":"11932983166720958399","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"11932983166720958399","cumulativeAtTargetRootRaw":"11932983166720958399","targetRootProof":["0x1e17510c9018ec36b8c2f284ed157855331b4e237ade37d502a0175cd5e1a7b1","0xd6b25a9dcda4414984830b5466924b2eaa6c177c4a09749d89c36a346bdc8c0b","0x2d28e1dfb7f3dd761854d605248c3fa5584cb1502d68145522d1e4efd71d30f5","0x14f6ca8bccb2f1331cac67d02d6708f319a32dd3e3400ce2d53a9e21f583c7ff","0x4a4994821801e0c42753f8795eeb750bd498aa94b27c2de2ba8b9b03293c1cba","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x0c599bf4b3db0f1089e095214e22ef1f3d75819b","campaignsAllocatedRaw":"3458171312365539562","cumulativeBeforeRaw":"26917620215293922843","cumulativeAfterRaw":"30375791527659462405","cumulativeAtTargetRootRaw":"30375791527659462405","targetRootProof":["0x456c8502c2f6ab11923a224d3d9802492d1165d6c0cee6eb8c0d64ac3e8bdf2e","0x4893ab96eb2f4569012c105d31fe87427e86a4ece40da7be0f1bc316a1d6ed7a","0x1244f85f0bdbb0035b1f9e92682281fd58b540b7babe515eb689c66167af1fe3","0x29c795a6902f7660cd1046cc476e5c1b0ab75033e94197690911a157c85a547c","0xa4ef0b10e7ce37fff8812be8f5b18de8338d3fcaa08d436c2f5e178b53a9bf56","0x974999b05f3a5032f5b03867cda64c0909a21b4fa56ac36f130c93177966362b","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x114b2f6cfff22d1ab14b94de3dc869816af2617b","campaignsAllocatedRaw":"9215324903996914599","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"9215324903996914599","cumulativeAtTargetRootRaw":"9215324903996914599","targetRootProof":["0x1ae20cda6d0fd3968da08f7805f029bc841db9bfdd20f20a83954bcd7e7f5f38","0x3ae221f94ec3dd30670c3371f6bc04203706c9f598097d1e2871979287a9d36e","0x3bae01972aeeb78145d25bc67415a0a1659adc0e28ccaf718539a3cd4c239ac4","0x14f6ca8bccb2f1331cac67d02d6708f319a32dd3e3400ce2d53a9e21f583c7ff","0x4a4994821801e0c42753f8795eeb750bd498aa94b27c2de2ba8b9b03293c1cba","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x140babc00370d3954fccdf0be68492d87a831eb6","campaignsAllocatedRaw":"2066026000456044999","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"2066026000456044999","cumulativeAtTargetRootRaw":"2066026000456044999","targetRootProof":["0x9fe63e57aceb35e40a5f555f1065ad54b9e697cd9760bda42022367e1a478895","0x114d7317214d3036b9488a71a0d404b339dd6ff59a4947a3c5afb2bd359a13be","0xae8ccbd296f4b4962eca1652d8d9799a86f3dce6f399a78b1af060d02f55f0c3","0x321e7c622a7be840554cb37b0081b6b40f15dad887e7ed831a7b1acdca023dd3","0x2ac5a1bb379274c8346f89601e358fe0e615a01cafd41190ec00295be48d9765","0x5e28347137c9ae7dab7eec2557a012f0555a234b39ab19f2bfa5c49326e401f7","0xfe17e3ac2f8a782948d8d3da8be4d80f66a14567b31128a88bff069887241ade","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x16ef04260ca959c75bd8d4247874517c9cf74d12","campaignsAllocatedRaw":"3161858867886445314","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"3161858867886445314","cumulativeAtTargetRootRaw":"3161858867886445314","targetRootProof":["0x3f7f5f28e57486863afb7f436fcf74a6b7e1e4958537e931b2b75ad59b1f4779","0x9f614f6f0f3ea728e7eab242d8202c5c6f6dfbfb92b0d33261be1f7d878dbb96","0x44c1a636219d453550f164c7740637c6ac0d9362cfbebbcd556c5477eb4c715c","0x7dbe9f35c5cf44175328ea18d9e63adcc963351519e41bfa3e161d09b67eaee5","0x2ac5a1bb379274c8346f89601e358fe0e615a01cafd41190ec00295be48d9765","0x5e28347137c9ae7dab7eec2557a012f0555a234b39ab19f2bfa5c49326e401f7","0xfe17e3ac2f8a782948d8d3da8be4d80f66a14567b31128a88bff069887241ade","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x27ce87b944e744eb415168d4da672a2135c4b06e","campaignsAllocatedRaw":"3769571719225909055","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"3769571719225909055","cumulativeAtTargetRootRaw":"3769571719225909055","targetRootProof":["0xa2947b7a46ad061cff3bed786e0b18d7f03a712f8c360718def31a68ce2ec524","0xa2d76dc7e55210c89bf579548227c57a38daccd800a98fb29a8a9c81ee042d27","0x575d2136b1f99338aacbde95d15039439250cc60115d947746cce7266b25cda3","0x50a7688c8f3eee4a834bc0a1fd66bfe2f05db1900d357360a4e7fb8129ec6222","0x4a4994821801e0c42753f8795eeb750bd498aa94b27c2de2ba8b9b03293c1cba","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x6cafe7d50b15334aba3b345e431a66e6c2f365f6","campaignsAllocatedRaw":"9668351031306762052","cumulativeBeforeRaw":"147945189650214017265","cumulativeAfterRaw":"157613540681520779317","cumulativeAtTargetRootRaw":"157613540681520779317","targetRootProof":["0xaba5cdbc6ac5aae6c371ade09235e28c27ed874ca776c0f69ea37176f31c2de5","0x3eff62b6895a7e6666d9b4ead29e69e1c29da7acb3f8f4ae31e1178d58108b87","0x4704d35e0b4d2d649f235039e146631b03abd44bb99571d2413de8269061a15c","0xb69c9213d18d14c8fcbeb45b97b62c7b4d50cf89329586d6bb8f8073c8f6d1df","0xa1209274b8d56bdd3e76ca1eb7475f91af777cc6f6732d4e36f2c4f48566b885","0x974999b05f3a5032f5b03867cda64c0909a21b4fa56ac36f130c93177966362b","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x915762adf03ddf5cfb10ac32e881c52ed0f59897","campaignsAllocatedRaw":"1509172677274688806","cumulativeBeforeRaw":"34599638426665697477","cumulativeAfterRaw":"36108811103940386283","cumulativeAtTargetRootRaw":"36108811103940386283","targetRootProof":["0xf1d038023c589d2625784925247d92c2d8cb214d0025ccf8effe0e4ac0c1fcc0","0xc01d62a7b13fdba151f63471b89cb1fa2f6e2c56319ee33f2ed5cb35797005d4","0x2d28e1dfb7f3dd761854d605248c3fa5584cb1502d68145522d1e4efd71d30f5","0x14f6ca8bccb2f1331cac67d02d6708f319a32dd3e3400ce2d53a9e21f583c7ff","0x4a4994821801e0c42753f8795eeb750bd498aa94b27c2de2ba8b9b03293c1cba","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0x9966956efc8576cf50dc7d21d712b258afd39b29","campaignsAllocatedRaw":"338004332072952143","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"338004332072952143","cumulativeAtTargetRootRaw":"338004332072952143","targetRootProof":["0xa6ca149422888d29ea886266e132be2f40d42bfc0dca5ccbb3be985286bac7ce","0x2b1c929e2a28c8cd2de257b32701b2f018903b7024656f15727e2d8f1064a57a","0xf29671ccf6ba5c4bd920e8e26bed3e1047f051d61a8d76257cf14d885f7b405e","0x7939de00919aa898b641eb80935770ceb5281ff184a2ad6f2032d6cd08eca188","0x62f46ecccd8e3bcde7a48481a6f1decd5ba0cfb9121ec4c6d9073e35032267b4","0x5e28347137c9ae7dab7eec2557a012f0555a234b39ab19f2bfa5c49326e401f7","0xfe17e3ac2f8a782948d8d3da8be4d80f66a14567b31128a88bff069887241ade","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0xa8fc3e362f3e5d3b03a0f670703a00518f950a7e","campaignsAllocatedRaw":"2448093488012032425","cumulativeBeforeRaw":"45506124649154271925","cumulativeAfterRaw":"47954218137166304350","cumulativeAtTargetRootRaw":"47954218137166304350","targetRootProof":["0x563c7befbb2b4254739b43e4668d952d2f2aa5809e7876ce8fee50ad2e385305","0x9d64fde9678afc5bda3604f38f65a3884921c8f67987f89d35c96a07017f1464","0x44c1a636219d453550f164c7740637c6ac0d9362cfbebbcd556c5477eb4c715c","0x7dbe9f35c5cf44175328ea18d9e63adcc963351519e41bfa3e161d09b67eaee5","0x2ac5a1bb379274c8346f89601e358fe0e615a01cafd41190ec00295be48d9765","0x5e28347137c9ae7dab7eec2557a012f0555a234b39ab19f2bfa5c49326e401f7","0xfe17e3ac2f8a782948d8d3da8be4d80f66a14567b31128a88bff069887241ade","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0xc9b7411fd802bdfa055cfd469d1aecac2ac64394","campaignsAllocatedRaw":"524449796696310502","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"524449796696310502","cumulativeAtTargetRootRaw":"524449796696310502","targetRootProof":["0x94351afdc5ea171308b98e74038ebdacfc8859fa62d8a050018df8277eab5b1b","0x013778e77369267f23be76c981dd0db5854fa321d69a5134e4c009eaa7e67d97","0xf29671ccf6ba5c4bd920e8e26bed3e1047f051d61a8d76257cf14d885f7b405e","0x7939de00919aa898b641eb80935770ceb5281ff184a2ad6f2032d6cd08eca188","0x62f46ecccd8e3bcde7a48481a6f1decd5ba0cfb9121ec4c6d9073e35032267b4","0x5e28347137c9ae7dab7eec2557a012f0555a234b39ab19f2bfa5c49326e401f7","0xfe17e3ac2f8a782948d8d3da8be4d80f66a14567b31128a88bff069887241ade","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0xc9c80e8481c2b6f979afc155bc3f979cfad19c56","campaignsAllocatedRaw":"299148006154037748318","cumulativeBeforeRaw":"8986970408404150763826","cumulativeAfterRaw":"9286118414558188512144","cumulativeAtTargetRootRaw":"9286118414558188512144","targetRootProof":["0xd2aa94b345e96a572200afdf045236511dab6b74a6a724efecc815a3ece49207","0x88de3e4724324829603eccc8ff89a4eb8d64e32f7bf157a92ff9f705c3ab342f","0x369d615ff3c2230b77320256080b9a8d0e990ad5e41fbbecf458fd965abd0843","0xe2cc661ff5ee94f29811299ca3237a85d4e0ad47545380338584b98a6d1912c6","0x08d19cfbe31d0c64a4418fc6feb4e22f3af3619b2770fec05fbeef73c669683b","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0xdde618e658ecce50809ce03913f65dd18e11ce64","campaignsAllocatedRaw":"5613340449329939309","cumulativeBeforeRaw":"64213958953958103933","cumulativeAfterRaw":"69827299403288043242","cumulativeAtTargetRootRaw":"69827299403288043242","targetRootProof":["0x57a4e8908f1eab1eb8d42eea4575bb3b739877c9f404d0fcc4fe6875b68b3073","0xc47355882552a606346f72176a250203a2af6473fcc08f2edfdbca1498ee24d0","0x369d615ff3c2230b77320256080b9a8d0e990ad5e41fbbecf458fd965abd0843","0xe2cc661ff5ee94f29811299ca3237a85d4e0ad47545380338584b98a6d1912c6","0x08d19cfbe31d0c64a4418fc6feb4e22f3af3619b2770fec05fbeef73c669683b","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]},{"account":"0xf7c40919a01780fe6f477c1535de0a969854bb48","campaignsAllocatedRaw":"6375557783258251712","cumulativeBeforeRaw":"0","cumulativeAfterRaw":"6375557783258251712","cumulativeAtTargetRootRaw":"6375557783258251712","targetRootProof":["0x113957ef8967829d743e3869400720f41af07aed07c9c7b715a606998ecdc641","0xdb3c9dcce2a53d500f48433207670f1714884ce209473d6a4e13c7b6c36b23c6","0x575d2136b1f99338aacbde95d15039439250cc60115d947746cce7266b25cda3","0x50a7688c8f3eee4a834bc0a1fd66bfe2f05db1900d357360a4e7fb8129ec6222","0x4a4994821801e0c42753f8795eeb750bd498aa94b27c2de2ba8b9b03293c1cba","0x2e84344ce73a3730dd934dcf43ac0b6bd8d592d647f5cb4619676b7111ec13be","0xb1f33fee2e517a6106d75bc4ef66c6463244484c7506a56420bfd2ef6f66a32d","0xb47a189265cd4449f402a37dfb919ec6917beac428876f63368cdad933b70fa4","0x22776d3f0fc9da732d816a8301901f9eb33bc49a9085acc06d793d25f2ec48cb","0xc5eee1e290816bc251c1dbc2a2621b2efddfadad38216e47aed17b325ea9a247","0x3f49fa89927356d856aa231ac8d984fde70d37a7078c8bdf83aec43105207caf","0xa2d6a4ca13f69666452f9c34f15fec20edb53df29fef8a3d5d1929ba0d833f25"]}],"campaigns":[{"id":2593,"vault":"0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad","gauge":"0x07a01471fa544d9c6531b631e6a96a79a9ad05e9","token":"0x6440f144b7e50d6a8439336510312d2f54beb01d","start":1795023887,"end":1795046483,"durationSeconds":22596,"fundedRaw":"10162926236299420784","rateRawPerSecond":"449766606315251","allocatedRaw":"10162926236299411596","allocatedTokens":"10.162926236299411596","recipients":5,"integerRateRemainderRaw":"9188","publishedWeightsAndPayoutArithmeticVerified":true,"users":[{"user":"0xC9c80e8481c2B6f979AFc155bc3F979cfAD19c56","balance":"21133512224284004124835192274280155035734","share":"93.527669","amount":"9505148072638088280"},{"user":"0x6cafE7d50b15334aba3b345e431a66E6c2f365f6","balance":"698297220890981016436680297319099654278","share":"3.090357","amount":"314070771239507728"},{"user":"0xDDe618E658ECcE50809ce03913f65dD18E11cE64","balance":"405423843527145369712695681361668331428","share":"1.794228","amount":"182346106222489515"},{"user":"0x0c599bf4B3Db0f1089E095214e22Ef1F3D75819b","balance":"249766626074124590642383869782361954904","share":"1.105357","amount":"112336687780169301"},{"user":"0x915762adF03ddF5cFB10Ac32e881c52eD0f59897","balance":"109000085223744653329724423834653531256","share":"0.482386","amount":"49024598419156772"}]},{"id":2594,"vault":"0x543d3ee7cfa494e8ffcc18364d130c8eed4b90ad","gauge":"0x07a01471fa544d9c6531b631e6a96a79a9ad05e9","token":"0x6440f144b7e50d6a8439336510312d2f54beb01d","start":1795046543,"end":1795110167,"durationSeconds":63624,"fundedRaw":"349080457011333255452","rateRawPerSecond":"5486616009860009","allocatedRaw":"349080457011333212616","allocatedTokens":"349.080457011333212616","recipients":15,"integerRateRemainderRaw":"42836","publishedWeightsAndPayoutArithmeticVerified":true,"users":[{"user":"0xC9c80e8481c2B6f979AFc155bc3F979cfAD19c56","balance":"52790801754830637455610593569626287662344","share":"82.973094","amount":"289642858081399660038"},{"user":"0x07d398f9cbf0f07DB51ae4c4105F38824821Aa48","balance":"2174925882415712803321392957533243714370","share":"3.418404","amount":"11932983166720958399"},{"user":"0x6cafE7d50b15334aba3b345e431a66E6c2f365f6","balance":"1704927088620136167572352194943190766676","share":"2.679691","amount":"9354280260067254324"},{"user":"0x114b2F6Cfff22D1Ab14b94de3DC869816Af2617b","balance":"1679600848216101557929539385254091253675","share":"2.639885","amount":"9215324903996914599"},{"user":"0xF7c40919A01780fe6F477c1535DE0a969854bB48","balance":"1162020045106259125602490939319259798910","share":"1.826386","amount":"6375557783258251712"},{"user":"0xDDe618E658ECcE50809ce03913f65dD18E11cE64","balance":"989862300067545843097863591661220681976","share":"1.555800","amount":"5430994343107449794"},{"user":"0x27ce87b944E744Eb415168D4Da672A2135C4B06e","balance":"687048576472565887401687929777319634448","share":"1.079857","amount":"3769571719225909055"},{"user":"0x0c599bf4B3Db0f1089E095214e22Ef1F3D75819b","balance":"609817530254088059820004038611317621968","share":"0.958470","amount":"3345834624585370261"},{"user":"0x16ef04260Ca959C75bd8d4247874517c9cF74d12","balance":"576285794778468531004723935613248764830","share":"0.905767","amount":"3161858867886445314"},{"user":"0xA8fC3E362F3E5d3B03A0f670703A00518F950A7e","balance":"446193698194398572548522105064097899552","share":"0.701297","amount":"2448093488012032425"},{"user":"0x140babc00370d3954fCCdf0Be68492d87a831Eb6","balance":"376557425696127668034924014763378585072","share":"0.591848","amount":"2066026000456044999"},{"user":"0x915762adF03ddF5cFB10Ac32e881c52eD0f59897","balance":"266129081428606769905383889086738436752","share":"0.418284","amount":"1460148078855532034"},{"user":"0xC9B7411fd802BDfA055cFD469D1aeCAc2Ac64394","balance":"95587115218892781520202998876724836885","share":"0.150237","amount":"524449796696310502"},{"user":"0x9966956efC8576CF50dC7D21D712B258Afd39b29","balance":"61605246560999316684236841990125263180","share":"0.096827","amount":"338004332072952143"},{"user":"0x01E246880e3d59f1af2F404a456026af5577b4A8","balance":"2637612139453558799322254885142601193","share":"0.004145","amount":"14471564992127017"}]}]}`
