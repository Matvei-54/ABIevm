package main

import (
	"context"
	"io"
	"testing"
)

func TestInvalidCheckDoesNotMakeRequests(t *testing.T) {
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	if err := run(context.Background(), nil, io.Discard, io.Discard); err == nil {
		t.Fatal("missing endpoint was accepted")
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "https://must-not-connect.invalid")
	for _, args := range [][]string{{"-limit", "4"}, {"-from", "1", "-to", "101"}, {"-from", "2", "-to", "1"}, {"unexpected"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}
