// stakedao-check runs a bounded manual mainnet check; normal go test never runs it.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao"
	dto "ABIevm/internal/service/stakedao/dto"
	"github.com/ethereum/go-ethereum/common"
)

// Manual sample candidates from gaugesMainFromController in the user's original
// stakeDaoService.go (commit ec418eb). Every registration is read onchain; this
// list is only for the diagnostic command, never production pool discovery.
const sampleGauges = "0xbE6228125448E40F5DCBB2DbdF7fF3F5721000B4,0x3285987Bbd0b695Ac48DC8571F519065B50aBA07,0x326b6d1a26Fd04afAa349340022076b5a2eb0c7C"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		// Transport errors can include a provider URL with its access token.
		message := err.Error()
		if endpoint := os.Getenv("STAKEDAO_ETHEREUM_RPC_URL"); endpoint != "" {
			message = strings.ReplaceAll(message, endpoint, "[RPC endpoint]")
		}
		message = regexp.MustCompile(`https?://[^\s"<>]+`).ReplaceAllString(message, "[URL hidden]")
		fmt.Fprintln(os.Stderr, "StakeDAO check failed:", message)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output, diagnostics io.Writer) error {
	flags := flag.NewFlagSet("stakedao-check", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	limit := flags.Int("limit", 1, "maximum pools to check (1–3)")
	gaugeList := flags.String("gauges", sampleGauges, "1–3 gauge addresses, comma-separated; checked against current Controller state")
	from := flags.Uint64("from", 0, "optional first registration block; requires -to")
	to := flags.Uint64("to", 0, "optional last registration block; requires -from; at most 100 blocks")
	filename := flags.String("output", "", "save JSON to this file only after a successful check")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if strings.TrimSpace(os.Getenv("STAKEDAO_ETHEREUM_RPC_URL")) == "" {
		return fmt.Errorf("set STAKEDAO_ETHEREUM_RPC_URL in this terminal before running the check")
	}
	if *limit < 1 || *limit > 3 {
		return fmt.Errorf("use -limit 1..3")
	}
	windowFlags, explicitGauges := 0, false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "from" || f.Name == "to" {
			windowFlags++
		}
		if f.Name == "gauges" {
			explicitGauges = true
		}
	})
	if windowFlags > 0 && (windowFlags != 2 || explicitGauges || *from > *to || *to-*from >= 100) {
		return fmt.Errorf("use both -from and -to for a window of at most 100 blocks, or use -gauges")
	}
	var gauges []common.Address
	if windowFlags == 0 {
		seen := make(map[common.Address]bool)
		for _, text := range strings.Split(*gaugeList, ",") {
			text = strings.TrimSpace(text)
			if !common.IsHexAddress(text) {
				return fmt.Errorf("each gauge must be a valid Ethereum address")
			}
			gauge := common.HexToAddress(text)
			if gauge == (common.Address{}) || seen[gauge] {
				return fmt.Errorf("gauge addresses must be nonzero and unique")
			}
			seen[gauge] = true
			gauges = append(gauges, gauge)
		}
		if len(gauges) > 3 {
			return fmt.Errorf("select at most three gauges")
		}
		if len(gauges) > *limit {
			gauges = gauges[:*limit]
		}
	}
	service := stakedao.NewStakeDAOService(prices.NewTokenPriceService(prices.NewMemoryStore(), prices.NewLlamaSource(nil)))
	var result *dto.StakeDAOPoolsResponse
	var err error
	fmt.Fprintln(diagnostics, "At most 5 RPC requests/sec and 300 attempts; CU cost depends on the provider.")
	if windowFlags == 0 {
		fmt.Fprintf(diagnostics, "Checking %d explicit gauge(s) using current Controller state; no log scan or historical eth_getCode.\n", len(gauges))
		result, err = service.GetPoolsForGauges(ctx, "ethereum", gauges, *limit)
	} else {
		fmt.Fprintf(diagnostics, "Checking up to %d pools from registration blocks %d–%d.\n", *limit, *from, *to)
		result, err = service.GetPoolsSample(ctx, "ethereum", *from, *to, *limit)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(diagnostics, "Loaded %d pool(s) at block %d; %d diagnostic issue(s).\n", len(result.Pools), result.BlockNumber, len(result.Errors))
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return err
	}
	if *filename == "" {
		_, err := output.Write(buffer.Bytes())
		return err
	}
	return saveResult(*filename, buffer.Bytes(), diagnostics)
}

func saveResult(filename string, data []byte, diagnostics io.Writer) error {
	path, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".stakedao-check-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}
	fmt.Fprintln(diagnostics, "JSON saved:", path)
	return nil
}
