package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidCheckDoesNotMakeRequests(t *testing.T) {
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	if err := run(context.Background(), nil, io.Discard, io.Discard); err == nil {
		t.Fatal("missing endpoint was accepted")
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "https://must-not-connect.invalid")
	for _, args := range [][]string{{"-limit", "6"}, {"-from", "1", "-to", "101"}, {"-from", "2", "-to", "1"}, {"-from", "1"}, {"-gauges", "invalid"}, {"-gauges", "0x0000000000000000000000000000000000000000"}, {"unexpected"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
	for _, args := range [][]string{{"-diagnostics"}, {"-diagnostics", "-limit", "3"}, {"-diagnostics", "-limit", "5", "-gauges", "invalid"}, {"-diagnostics", "-limit", "2", "-from", "1", "-to", "2"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid diagnostics selection accepted: %v", args)
		}
	}
	for _, args := range [][]string{{"-block", "1"}, {"-strict-prices"}, {"-diagnostics", "-limit", "2", "-block", "0"}, {"-diagnostics", "-limit", "2", "-block", "-1"}} {
		if err := run(context.Background(), args, io.Discard, io.Discard); err == nil {
			t.Fatalf("invalid snapshot arguments accepted: %v", args)
		}
	}
}

func TestFileIsSavedOnlyAfterSuccessAndCanBeReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.json")
	want := []byte("{\"pools\": []}\n")
	if err := saveResult(path, want, io.Discard); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STAKEDAO_ETHEREUM_RPC_URL", "")
	if err := run(context.Background(), []string{"-output", path}, io.Discard, io.Discard); err == nil {
		t.Fatal("missing endpoint was accepted")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("failed check modified previous output: %s %v", got, err)
	}
	want = []byte("{\"pools\": [], \"network\": \"ethereum\"}\n")
	if err := saveResult(path, want, io.Discard); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("could not replace output: %s %v", got, err)
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".stakedao-check-*.tmp"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary result files were left behind")
	}
}
