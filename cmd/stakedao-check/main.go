// stakedao-check runs a bounded manual mainnet check; normal go test never runs it.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"

	"ABIevm/internal/service/prices"
	"ABIevm/internal/service/stakedao"
)

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
	from := flags.Uint64("from", 23225415, "first block of the registration window")
	to := flags.Uint64("to", 23225514, "last block of the registration window (at most 100 blocks)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if strings.TrimSpace(os.Getenv("STAKEDAO_ETHEREUM_RPC_URL")) == "" {
		return fmt.Errorf("set STAKEDAO_ETHEREUM_RPC_URL in this terminal before running the check")
	}
	if *limit < 1 || *limit > 3 || *from > *to || *to-*from >= 100 {
		return fmt.Errorf("use -limit 1..3 and a registration window of at most 100 blocks")
	}
	fmt.Fprintf(diagnostics, "Checking up to %d pools from registration blocks %d–%d; at most 5 RPC requests/sec and 300 RPC attempts. CU cost depends on the provider.\n", *limit, *from, *to)
	service := stakedao.NewStakeDAOService(prices.NewTokenPriceService(prices.NewMemoryStore(), prices.NewLlamaSource(nil)))
	result, err := service.GetPoolsSample(ctx, "ethereum", *from, *to, *limit)
	if err != nil {
		return err
	}
	fmt.Fprintf(diagnostics, "Loaded %d pool(s) at block %d; %d diagnostic issue(s). Fewer pools than requested may be available in this window.\n", len(result.Pools), result.BlockNumber, len(result.Errors))
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
