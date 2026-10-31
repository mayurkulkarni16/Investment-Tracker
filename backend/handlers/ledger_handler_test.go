package handlers

import "testing"

func TestLedgerCSVFormulaNeutralization(t *testing.T) {
	for _, value := range []string{"=1+1", " +SUM(A1)", "\t@formula", "-cmd"} {
		if got := ledgerCSVCell(value); got != "'"+value {
			t.Fatalf("formula not neutralized: %q", got)
		}
	}
	for _, value := range []string{"", "  ", "HDFC", "2026-10-21", "1234"} {
		if got := ledgerCSVCell(value); got != value {
			t.Fatalf("ordinary value changed: %q", got)
		}
	}
}
