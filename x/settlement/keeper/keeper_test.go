package keeper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VelaFlux28/ftg-chain/x/settlement/types"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data"), 0755)
	return dir
}

func TestCreateDepositIntent(t *testing.T) {
	k := NewKeeper(tempDir(t))

	dep, err := k.CreateDepositIntent("w1", "user1", 10_000_000, types.PaymentMethodStripe)
	if err != nil {
		t.Fatalf("CreateDepositIntent failed: %v", err)
	}
	if dep.Status != "pending" {
		t.Fatalf("expected pending, got %s", dep.Status)
	}
	if dep.Amount != 10_000_000 {
		t.Fatalf("expected 10M, got %d", dep.Amount)
	}
}

func TestCreateDepositBelowMinimum(t *testing.T) {
	k := NewKeeper(tempDir(t))

	_, err := k.CreateDepositIntent("w1", "user1", 100, types.PaymentMethodStripe)
	if err == nil {
		t.Fatal("should reject deposit below minimum")
	}
}

func TestCreateDepositDisabledMethod(t *testing.T) {
	k := NewKeeper(tempDir(t))

	_, err := k.CreateDepositIntent("w1", "user1", 10_000_000, types.PaymentMethodBankWire)
	if err == nil {
		t.Fatal("should reject disabled payment method")
	}
}

func TestConfirmDeposit(t *testing.T) {
	k := NewKeeper(tempDir(t))

	dep, _ := k.CreateDepositIntent("w1", "user1", 10_000_000, types.PaymentMethodStripe)
	confirmed, err := k.ConfirmDeposit(dep.ID, "stripe_pi_123")
	if err != nil {
		t.Fatalf("ConfirmDeposit failed: %v", err)
	}
	if confirmed.Status != "confirmed" {
		t.Fatalf("expected confirmed, got %s", confirmed.Status)
	}
	if confirmed.PaymentRef != "stripe_pi_123" {
		t.Fatalf("expected stripe_pi_123, got %s", confirmed.PaymentRef)
	}
	if k.TotalDeposited != 10_000_000 {
		t.Fatalf("expected total deposited 10M, got %d", k.TotalDeposited)
	}
}

func TestDoubleConfirmDeposit(t *testing.T) {
	k := NewKeeper(tempDir(t))

	dep, _ := k.CreateDepositIntent("w1", "user1", 10_000_000, types.PaymentMethodStripe)
	k.ConfirmDeposit(dep.ID, "ref1")
	_, err := k.ConfirmDeposit(dep.ID, "ref2")
	if err == nil {
		t.Fatal("should reject double confirmation")
	}
}

func TestEscrowLifecycle(t *testing.T) {
	k := NewKeeper(tempDir(t))

	// Create escrow
	escrow, err := k.CreateEscrow("trade1", "buyer_w", "seller_w", 25_000_000, types.PaymentMethodStripe)
	if err != nil {
		t.Fatalf("CreateEscrow failed: %v", err)
	}
	if escrow.Status != types.StatusPending {
		t.Fatalf("expected pending, got %s", escrow.Status)
	}
	if k.ActiveEscrows != 1 {
		t.Fatalf("expected 1 active escrow, got %d", k.ActiveEscrows)
	}

	// Fund escrow
	funded, err := k.FundEscrow(escrow.ID, "stripe_pi_456")
	if err != nil {
		t.Fatalf("FundEscrow failed: %v", err)
	}
	if funded.Status != types.StatusFunded {
		t.Fatalf("expected funded, got %s", funded.Status)
	}

	// Execute settlement
	record, err := k.ExecuteSettlement(escrow.ID, 5_000_000, 100)
	if err != nil {
		t.Fatalf("ExecuteSettlement failed: %v", err)
	}
	if record.FTGAmount != 5_000_000 {
		t.Fatalf("expected 5M uftg, got %d", record.FTGAmount)
	}
	if record.USDCAmount != 25_000_000 {
		t.Fatalf("expected 25M uusdc, got %d", record.USDCAmount)
	}
	if k.ActiveEscrows != 0 {
		t.Fatalf("expected 0 active escrows after settlement, got %d", k.ActiveEscrows)
	}
	if k.TotalSettledUSDC != 25_000_000 {
		t.Fatalf("expected total settled 25M, got %d", k.TotalSettledUSDC)
	}
}

func TestEscrowRefund(t *testing.T) {
	k := NewKeeper(tempDir(t))

	escrow, _ := k.CreateEscrow("trade2", "buyer_w", "seller_w", 10_000_000, types.PaymentMethodStripe)
	k.FundEscrow(escrow.ID, "ref")

	err := k.RefundEscrow(escrow.ID, "buyer cancelled")
	if err != nil {
		t.Fatalf("RefundEscrow failed: %v", err)
	}

	e, _ := k.GetEscrow(escrow.ID)
	if e.Status != types.StatusRefunded {
		t.Fatalf("expected refunded, got %s", e.Status)
	}
}

func TestWithdrawalFlow(t *testing.T) {
	k := NewKeeper(tempDir(t))

	req, err := k.CreateWithdrawalRequest("w1", "user1", 50_000_000, types.PaymentMethodStablecoin, "0xabc123")
	if err != nil {
		t.Fatalf("CreateWithdrawalRequest failed: %v", err)
	}
	if req.Status != "pending" {
		t.Fatalf("expected pending, got %s", req.Status)
	}

	processed, err := k.ProcessWithdrawal(req.ID)
	if err != nil {
		t.Fatalf("ProcessWithdrawal failed: %v", err)
	}
	if processed.Status != "completed" {
		t.Fatalf("expected completed, got %s", processed.Status)
	}
}

func TestSettlementStats(t *testing.T) {
	k := NewKeeper(tempDir(t))

	stats := k.GetStats()
	if stats["total_settlements"].(int) != 0 {
		t.Fatalf("expected 0 settlements, got %v", stats["total_settlements"])
	}
	if stats["stripe_enabled"].(bool) != true {
		t.Fatal("stripe should be enabled by default")
	}
	if stats["bank_wire_enabled"].(bool) != false {
		t.Fatal("bank wire should be disabled by default")
	}
}

func TestPersistence(t *testing.T) {
	dir := tempDir(t)

	k1 := NewKeeper(dir)
	k1.CreateDepositIntent("w1", "user1", 10_000_000, types.PaymentMethodStripe)
	k1.CreateEscrow("trade1", "b", "s", 5_000_000, types.PaymentMethodStripe)

	// Load from same dir
	k2 := NewKeeper(dir)
	if len(k2.Deposits) != 1 {
		t.Fatalf("expected 1 deposit after reload, got %d", len(k2.Deposits))
	}
	if len(k2.Escrows) != 1 {
		t.Fatalf("expected 1 escrow after reload, got %d", len(k2.Escrows))
	}
}
