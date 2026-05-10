package keeper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/VelaFlux28/ftg-chain/x/exchange/types"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data"), 0755)
	return dir
}

func TestCreateWallet(t *testing.T) {
	k := NewKeeper(tempDir(t))

	wallet, err := k.CreateWallet("user1", "My Trading Wallet")
	if err != nil {
		t.Fatalf("CreateWallet failed: %v", err)
	}
	if wallet.ID == "" {
		t.Fatal("wallet ID should not be empty")
	}
	if wallet.UserID != "user1" {
		t.Fatalf("expected user1, got %s", wallet.UserID)
	}
	if wallet.KYCTier != types.KYCTierNone {
		t.Fatalf("new wallet should have KYCTierNone")
	}

	// Duplicate should fail
	_, err = k.CreateWallet("user1", "Another Wallet")
	if err != types.ErrWalletExists {
		t.Fatalf("expected ErrWalletExists, got %v", err)
	}
}

func TestDepositAndWithdraw(t *testing.T) {
	k := NewKeeper(tempDir(t))

	wallet, _ := k.CreateWallet("user1", "Test")

	// Deposit
	err := k.DepositToWallet(wallet.ID, "uftg", 100_000_000)
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	w, _ := k.GetWallet(wallet.ID)
	if w.Balances["uftg"] != 100_000_000 {
		t.Fatalf("expected 100M uftg, got %d", w.Balances["uftg"])
	}

	// Withdraw
	err = k.WithdrawFromWallet(wallet.ID, "uftg", 50_000_000)
	if err != nil {
		t.Fatalf("withdraw failed: %v", err)
	}

	w, _ = k.GetWallet(wallet.ID)
	if w.Balances["uftg"] != 50_000_000 {
		t.Fatalf("expected 50M uftg, got %d", w.Balances["uftg"])
	}

	// Over-withdraw should fail
	err = k.WithdrawFromWallet(wallet.ID, "uftg", 100_000_000)
	if err != types.ErrInsufficientBalance {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestKYCBasicAutoApproval(t *testing.T) {
	k := NewKeeper(tempDir(t))

	wallet, _ := k.CreateWallet("user1", "Test")

	app := &types.KYCApplication{
		UserID:        "user1",
		WalletID:      wallet.ID,
		RequestedTier: types.KYCTierBasic,
		Email:         "user@example.com",
		Phone:         "+1234567890",
		Country:       "US",
	}

	result, err := k.SubmitKYC(app)
	if err != nil {
		t.Fatalf("SubmitKYC failed: %v", err)
	}

	// Basic tier from non-sanctioned country should auto-approve
	if result.Status != types.KYCStatusApproved {
		t.Fatalf("expected approved, got %s", result.Status)
	}

	// Wallet should now be tier 1
	w, _ := k.GetWallet(wallet.ID)
	if w.KYCTier != types.KYCTierBasic {
		t.Fatalf("expected KYCTierBasic, got %d", w.KYCTier)
	}
}

func TestKYCSanctionedCountryRejection(t *testing.T) {
	k := NewKeeper(tempDir(t))

	wallet, _ := k.CreateWallet("user2", "Test")

	app := &types.KYCApplication{
		UserID:        "user2",
		WalletID:      wallet.ID,
		RequestedTier: types.KYCTierBasic,
		Email:         "user@example.com",
		Country:       "KP", // North Korea
	}

	result, err := k.SubmitKYC(app)
	if err != nil {
		t.Fatalf("SubmitKYC failed: %v", err)
	}

	if result.Status != types.KYCStatusRejected {
		t.Fatalf("expected rejected for sanctioned country, got %s", result.Status)
	}

	// Wallet should be frozen
	w, _ := k.GetWallet(wallet.ID)
	if w.Active {
		t.Fatal("wallet should be frozen after sanctions hit")
	}
}

func TestPlaceOrderRequiresKYC(t *testing.T) {
	k := NewKeeper(tempDir(t))

	wallet, _ := k.CreateWallet("user1", "Test")
	k.DepositToWallet(wallet.ID, "uusdc", 1_000_000_000)

	// Should fail without KYC
	_, _, err := k.PlaceOrder(wallet.ID, "FTG-USDC", types.OrderSideBuy, types.OrderTypeLimit, 5_000_000, 1_000_000, 100)
	if err != types.ErrInsufficientKYC {
		t.Fatalf("expected ErrInsufficientKYC, got %v", err)
	}
}

func TestPlaceAndMatchOrders(t *testing.T) {
	k := NewKeeper(tempDir(t))

	// Create two wallets with KYC
	w1, _ := k.CreateWallet("seller1", "Seller")
	w2, _ := k.CreateWallet("buyer1", "Buyer")

	// Approve KYC for both
	app1 := &types.KYCApplication{UserID: "seller1", WalletID: w1.ID, RequestedTier: types.KYCTierBasic, Country: "US"}
	k.SubmitKYC(app1)
	app2 := &types.KYCApplication{UserID: "buyer1", WalletID: w2.ID, RequestedTier: types.KYCTierBasic, Country: "US"}
	k.SubmitKYC(app2)

	// Deposit funds
	k.DepositToWallet(w1.ID, "uftg", 10_000_000)    // 10 FTG
	k.DepositToWallet(w2.ID, "uusdc", 100_000_000)  // 100 USDC

	// Seller places ask at $5.00 (5_000_000 uusdc per 1_000_000 uftg)
	sellOrder, trades1, err := k.PlaceOrder(w1.ID, "FTG-USDC", types.OrderSideSell, types.OrderTypeLimit, 5_000_000, 5_000_000, 100)
	if err != nil {
		t.Fatalf("sell order failed: %v", err)
	}
	if len(trades1) != 0 {
		t.Fatal("sell order should not match (no bids yet)")
	}
	if sellOrder.Status != types.OrderStatusOpen {
		t.Fatalf("expected open, got %s", sellOrder.Status)
	}

	// Buyer places bid at $5.00 for 3 FTG
	buyOrder, trades2, err := k.PlaceOrder(w2.ID, "FTG-USDC", types.OrderSideBuy, types.OrderTypeLimit, 5_000_000, 3_000_000, 101)
	if err != nil {
		t.Fatalf("buy order failed: %v", err)
	}

	// Should match
	if len(trades2) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(trades2))
	}
	if trades2[0].Quantity != 3_000_000 {
		t.Fatalf("expected trade qty 3M, got %d", trades2[0].Quantity)
	}
	if buyOrder.Status != types.OrderStatusFilled {
		t.Fatalf("buy order should be filled, got %s", buyOrder.Status)
	}

	// Sell order should be partially filled
	so, _ := k.GetOrderFromMap(sellOrder.ID)
	if so.Status != types.OrderStatusPartial {
		t.Fatalf("sell order should be partial, got %s", so.Status)
	}
	if so.RemainingQty != 2_000_000 {
		t.Fatalf("expected 2M remaining, got %d", so.RemainingQty)
	}
}

func TestCancelOrder(t *testing.T) {
	k := NewKeeper(tempDir(t))

	w1, _ := k.CreateWallet("user1", "Test")
	app := &types.KYCApplication{UserID: "user1", WalletID: w1.ID, RequestedTier: types.KYCTierBasic, Country: "US"}
	k.SubmitKYC(app)
	k.DepositToWallet(w1.ID, "uftg", 10_000_000)

	order, _, _ := k.PlaceOrder(w1.ID, "FTG-USDC", types.OrderSideSell, types.OrderTypeLimit, 5_000_000, 5_000_000, 100)

	// Cancel
	err := k.CancelOrder(order.ID, w1.ID)
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	// Funds should be unlocked
	w, _ := k.GetWallet(w1.ID)
	if w.Locked["uftg"] != 0 {
		t.Fatalf("expected 0 locked, got %d", w.Locked["uftg"])
	}

	// Double cancel should fail
	err = k.CancelOrder(order.ID, w1.ID)
	if err != types.ErrOrderCancelled {
		t.Fatalf("expected ErrOrderCancelled, got %v", err)
	}
}

func TestSelfTradeProtection(t *testing.T) {
	k := NewKeeper(tempDir(t))

	w1, _ := k.CreateWallet("user1", "Test")
	app := &types.KYCApplication{UserID: "user1", WalletID: w1.ID, RequestedTier: types.KYCTierBasic, Country: "US"}
	k.SubmitKYC(app)
	k.DepositToWallet(w1.ID, "uftg", 10_000_000)
	k.DepositToWallet(w1.ID, "uusdc", 100_000_000)

	// Place sell
	k.PlaceOrder(w1.ID, "FTG-USDC", types.OrderSideSell, types.OrderTypeLimit, 5_000_000, 5_000_000, 100)

	// Place buy from same wallet — should NOT match (self-trade protection)
	buyOrder, trades, err := k.PlaceOrder(w1.ID, "FTG-USDC", types.OrderSideBuy, types.OrderTypeLimit, 5_000_000, 3_000_000, 101)
	if err != nil {
		t.Fatalf("buy order failed: %v", err)
	}
	if len(trades) != 0 {
		t.Fatal("self-trade should be prevented")
	}
	if buyOrder.Status != types.OrderStatusOpen {
		t.Fatalf("buy order should remain open, got %s", buyOrder.Status)
	}
}

func TestOrderBookSnapshot(t *testing.T) {
	k := NewKeeper(tempDir(t))

	w1, _ := k.CreateWallet("seller1", "Seller")
	w2, _ := k.CreateWallet("buyer1", "Buyer")
	app1 := &types.KYCApplication{UserID: "seller1", WalletID: w1.ID, RequestedTier: types.KYCTierBasic, Country: "US"}
	k.SubmitKYC(app1)
	app2 := &types.KYCApplication{UserID: "buyer1", WalletID: w2.ID, RequestedTier: types.KYCTierBasic, Country: "US"}
	k.SubmitKYC(app2)

	k.DepositToWallet(w1.ID, "uftg", 100_000_000)
	k.DepositToWallet(w2.ID, "uusdc", 500_000_000)

	// Place multiple asks
	k.PlaceOrder(w1.ID, "FTG-USDC", types.OrderSideSell, types.OrderTypeLimit, 5_100_000, 10_000_000, 100)
	k.PlaceOrder(w1.ID, "FTG-USDC", types.OrderSideSell, types.OrderTypeLimit, 5_200_000, 20_000_000, 100)

	// Place multiple bids
	k.PlaceOrder(w2.ID, "FTG-USDC", types.OrderSideBuy, types.OrderTypeLimit, 4_900_000, 10_000_000, 100)
	k.PlaceOrder(w2.ID, "FTG-USDC", types.OrderSideBuy, types.OrderTypeLimit, 4_800_000, 15_000_000, 100)

	book := k.GetOrderBook("FTG-USDC", 10)
	if len(book.Asks) != 2 {
		t.Fatalf("expected 2 ask levels, got %d", len(book.Asks))
	}
	if len(book.Bids) != 2 {
		t.Fatalf("expected 2 bid levels, got %d", len(book.Bids))
	}

	// Best ask should be lowest
	if book.Asks[0].Price != 5_100_000 {
		t.Fatalf("best ask should be 5.10, got %d", book.Asks[0].Price)
	}
	// Best bid should be highest
	if book.Bids[0].Price != 4_900_000 {
		t.Fatalf("best bid should be 4.90, got %d", book.Bids[0].Price)
	}
}

func TestPersistence(t *testing.T) {
	dir := tempDir(t)

	// Create keeper and add data
	k1 := NewKeeper(dir)
	k1.CreateWallet("user1", "Test")
	k1.DepositToWallet("FTG-W-000001", "uftg", 50_000_000)

	// Create new keeper from same dir — should load state
	k2 := NewKeeper(dir)
	w, err := k2.GetWallet("FTG-W-000001")
	if err != nil {
		t.Fatalf("wallet should persist: %v", err)
	}
	if w.Balances["uftg"] != 50_000_000 {
		t.Fatalf("balance should persist, got %d", w.Balances["uftg"])
	}
}

// Helper to get order from internal map (for test assertions)
func (k *Keeper) GetOrderFromMap(orderID string) (*types.Order, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	o, ok := k.Orders[orderID]
	return o, ok
}
