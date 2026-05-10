package keeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/VelaFlux28/ftg-chain/x/settlement/types"
)

// Keeper manages USDC settlement state
type Keeper struct {
	mu sync.RWMutex

	Escrows      map[string]*types.EscrowAccount     `json:"escrows"`
	Settlements  []*types.SettlementRecord            `json:"settlements"`
	Deposits     map[string]*types.DepositIntent      `json:"deposits"`
	Withdrawals  map[string]*types.WithdrawalRequest  `json:"withdrawals"`
	Config       *types.SettlementConfig              `json:"config"`

	// Aggregate stats
	TotalSettledUSDC   int64 `json:"total_settled_usdc"`
	TotalFeesCollected int64 `json:"total_fees_collected"`
	TotalDeposited     int64 `json:"total_deposited"`
	TotalWithdrawn     int64 `json:"total_withdrawn"`
	ActiveEscrows      int   `json:"active_escrows"`

	// Sequences
	EscrowSeq     int64 `json:"escrow_seq"`
	SettlementSeq int64 `json:"settlement_seq"`
	DepositSeq    int64 `json:"deposit_seq"`
	WithdrawalSeq int64 `json:"withdrawal_seq"`

	filePath string
}

// NewKeeper creates or loads the settlement keeper
func NewKeeper(homeDir string) *Keeper {
	statePath := filepath.Join(homeDir, "data", "settlement_state.json")

	k := &Keeper{
		Escrows:     make(map[string]*types.EscrowAccount),
		Settlements: make([]*types.SettlementRecord, 0),
		Deposits:    make(map[string]*types.DepositIntent),
		Withdrawals: make(map[string]*types.WithdrawalRequest),
		Config:      types.DefaultSettlementConfig(),
		filePath:    statePath,
	}

	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, k)
	}

	return k
}

// ============================================================
// DEPOSIT FLOW
// ============================================================

// CreateDepositIntent creates a new deposit intent for a user
func (k *Keeper) CreateDepositIntent(walletID, userID string, amount int64, method types.PaymentMethod) (*types.DepositIntent, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if amount < k.Config.MinSettlementUSDC {
		return nil, fmt.Errorf("minimum deposit is %d uusdc ($%.2f)", k.Config.MinSettlementUSDC, float64(k.Config.MinSettlementUSDC)/1_000_000)
	}

	// Validate payment method is enabled
	if !k.isMethodEnabled(method) {
		return nil, fmt.Errorf("payment method %s is not enabled", method)
	}

	k.DepositSeq++
	intent := &types.DepositIntent{
		ID:            fmt.Sprintf("DEP-%06d", k.DepositSeq),
		WalletID:      walletID,
		UserID:        userID,
		Amount:        amount,
		PaymentMethod: method,
		Status:        "pending",
		CreatedAt:     time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().Add(time.Duration(k.Config.EscrowTimeoutHours) * time.Hour),
	}

	// Generate deposit address for crypto deposits
	if method == types.PaymentMethodStablecoin || method == types.PaymentMethodNobleIBC {
		intent.DepositAddress = fmt.Sprintf("ftg1deposit%s", intent.ID)
	}

	k.Deposits[intent.ID] = intent
	k.save()
	return intent, nil
}

// ConfirmDeposit confirms a deposit has been received
func (k *Keeper) ConfirmDeposit(depositID, paymentRef string) (*types.DepositIntent, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	intent, ok := k.Deposits[depositID]
	if !ok {
		return nil, fmt.Errorf("deposit %s not found", depositID)
	}

	if intent.Status != "pending" {
		return nil, fmt.Errorf("deposit %s is already %s", depositID, intent.Status)
	}

	if time.Now().UTC().After(intent.ExpiresAt) {
		intent.Status = "expired"
		k.save()
		return nil, fmt.Errorf("deposit %s has expired", depositID)
	}

	now := time.Now().UTC()
	intent.Status = "confirmed"
	intent.PaymentRef = paymentRef
	intent.ConfirmedAt = &now
	k.TotalDeposited += intent.Amount
	k.save()
	return intent, nil
}

// ============================================================
// WITHDRAWAL FLOW
// ============================================================

// CreateWithdrawalRequest creates a new withdrawal request
func (k *Keeper) CreateWithdrawalRequest(walletID, userID string, amount int64, method types.PaymentMethod, destinationRef string) (*types.WithdrawalRequest, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.isMethodEnabled(method) {
		return nil, fmt.Errorf("payment method %s is not enabled", method)
	}

	// Calculate fee
	fee := k.Config.WithdrawalFeeFlat + (amount * int64(k.Config.WithdrawalFeeBPS) / 10_000)
	if amount <= fee {
		return nil, fmt.Errorf("withdrawal amount must exceed fee of %d uusdc", fee)
	}

	k.WithdrawalSeq++
	req := &types.WithdrawalRequest{
		ID:             fmt.Sprintf("WDR-%06d", k.WithdrawalSeq),
		WalletID:       walletID,
		UserID:         userID,
		Amount:         amount,
		PaymentMethod:  method,
		Status:         "pending",
		DestinationRef: destinationRef,
		CreatedAt:      time.Now().UTC(),
	}

	k.Withdrawals[req.ID] = req
	k.save()
	return req, nil
}

// ProcessWithdrawal marks a withdrawal as completed
func (k *Keeper) ProcessWithdrawal(withdrawalID string) (*types.WithdrawalRequest, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	req, ok := k.Withdrawals[withdrawalID]
	if !ok {
		return nil, fmt.Errorf("withdrawal %s not found", withdrawalID)
	}

	if req.Status != "pending" {
		return nil, fmt.Errorf("withdrawal %s is already %s", withdrawalID, req.Status)
	}

	now := time.Now().UTC()
	req.Status = "completed"
	req.ProcessedAt = &now

	fee := k.Config.WithdrawalFeeFlat + (req.Amount * int64(k.Config.WithdrawalFeeBPS) / 10_000)
	k.TotalWithdrawn += req.Amount
	k.TotalFeesCollected += fee
	k.save()
	return req, nil
}

// ============================================================
// ESCROW / SETTLEMENT FLOW
// ============================================================

// CreateEscrow creates a new escrow for a trade
func (k *Keeper) CreateEscrow(tradeID, buyerWalletID, sellerWalletID string, amountUSDC int64, method types.PaymentMethod) (*types.EscrowAccount, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if amountUSDC < k.Config.MinSettlementUSDC {
		return nil, fmt.Errorf("minimum settlement is %d uusdc", k.Config.MinSettlementUSDC)
	}
	if amountUSDC > k.Config.MaxSettlementUSDC {
		return nil, fmt.Errorf("maximum settlement is %d uusdc", k.Config.MaxSettlementUSDC)
	}

	fee := amountUSDC * int64(k.Config.SettlementFeeBPS) / 10_000

	k.EscrowSeq++
	escrow := &types.EscrowAccount{
		ID:             fmt.Sprintf("ESC-%06d", k.EscrowSeq),
		TradeID:        tradeID,
		BuyerWalletID:  buyerWalletID,
		SellerWalletID: sellerWalletID,
		AmountUSDC:     amountUSDC,
		FeeUSDC:        fee,
		Status:         types.StatusPending,
		PaymentMethod:  method,
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(time.Duration(k.Config.EscrowTimeoutHours) * time.Hour),
	}

	k.Escrows[escrow.ID] = escrow
	k.ActiveEscrows++
	k.save()
	return escrow, nil
}

// FundEscrow marks an escrow as funded (USDC received)
func (k *Keeper) FundEscrow(escrowID, paymentRef string) (*types.EscrowAccount, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	escrow, ok := k.Escrows[escrowID]
	if !ok {
		return nil, fmt.Errorf("escrow %s not found", escrowID)
	}

	if escrow.Status != types.StatusPending {
		return nil, fmt.Errorf("escrow %s is %s, cannot fund", escrowID, escrow.Status)
	}

	if time.Now().UTC().After(escrow.ExpiresAt) {
		escrow.Status = types.StatusFailed
		k.ActiveEscrows--
		k.save()
		return nil, fmt.Errorf("escrow %s has expired", escrowID)
	}

	now := time.Now().UTC()
	escrow.Status = types.StatusFunded
	escrow.PaymentRef = paymentRef
	escrow.FundedAt = &now
	k.save()
	return escrow, nil
}

// ExecuteSettlement completes the trade — releases USDC to seller, records settlement
func (k *Keeper) ExecuteSettlement(escrowID string, ftgAmount, blockHeight int64) (*types.SettlementRecord, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	escrow, ok := k.Escrows[escrowID]
	if !ok {
		return nil, fmt.Errorf("escrow %s not found", escrowID)
	}

	if escrow.Status != types.StatusFunded {
		return nil, fmt.Errorf("escrow %s is %s, must be funded to execute", escrowID, escrow.Status)
	}

	now := time.Now().UTC()
	escrow.Status = types.StatusCompleted
	escrow.CompletedAt = &now
	k.ActiveEscrows--

	// Calculate price per FTG
	pricePerFTG := int64(0)
	if ftgAmount > 0 {
		pricePerFTG = (escrow.AmountUSDC * 1_000_000) / ftgAmount
	}

	k.SettlementSeq++
	record := &types.SettlementRecord{
		ID:             fmt.Sprintf("STL-%06d", k.SettlementSeq),
		EscrowID:       escrowID,
		TradeID:        escrow.TradeID,
		BuyerWalletID:  escrow.BuyerWalletID,
		SellerWalletID: escrow.SellerWalletID,
		FTGAmount:      ftgAmount,
		USDCAmount:     escrow.AmountUSDC,
		PricePerFTG:    pricePerFTG,
		FeeCollected:   escrow.FeeUSDC,
		PaymentMethod:  escrow.PaymentMethod,
		BlockHeight:    blockHeight,
		SettledAt:      now,
	}

	k.Settlements = append(k.Settlements, record)
	k.TotalSettledUSDC += escrow.AmountUSDC
	k.TotalFeesCollected += escrow.FeeUSDC
	k.save()
	return record, nil
}

// RefundEscrow returns USDC to the buyer
func (k *Keeper) RefundEscrow(escrowID, reason string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	escrow, ok := k.Escrows[escrowID]
	if !ok {
		return fmt.Errorf("escrow %s not found", escrowID)
	}

	if escrow.Status == types.StatusCompleted {
		return fmt.Errorf("cannot refund completed escrow %s", escrowID)
	}

	escrow.Status = types.StatusRefunded
	if escrow.Status == types.StatusFunded || escrow.Status == types.StatusPending {
		k.ActiveEscrows--
	}
	k.save()
	return nil
}

// ============================================================
// QUERIES
// ============================================================

// GetEscrow returns an escrow by ID
func (k *Keeper) GetEscrow(escrowID string) (*types.EscrowAccount, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	escrow, ok := k.Escrows[escrowID]
	if !ok {
		return nil, fmt.Errorf("escrow %s not found", escrowID)
	}
	return escrow, nil
}

// GetSettlements returns recent settlements
func (k *Keeper) GetSettlements(limit int) []*types.SettlementRecord {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if limit <= 0 || limit > len(k.Settlements) {
		limit = len(k.Settlements)
	}

	// Return most recent first
	start := len(k.Settlements) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*types.SettlementRecord, 0, limit)
	for i := len(k.Settlements) - 1; i >= start; i-- {
		result = append(result, k.Settlements[i])
	}
	return result
}

// GetDeposit returns a deposit by ID
func (k *Keeper) GetDeposit(depositID string) (*types.DepositIntent, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	dep, ok := k.Deposits[depositID]
	if !ok {
		return nil, fmt.Errorf("deposit %s not found", depositID)
	}
	return dep, nil
}

// GetUserDeposits returns all deposits for a user
func (k *Keeper) GetUserDeposits(userID string) []*types.DepositIntent {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var result []*types.DepositIntent
	for _, d := range k.Deposits {
		if d.UserID == userID {
			result = append(result, d)
		}
	}
	return result
}

// GetUserWithdrawals returns all withdrawals for a user
func (k *Keeper) GetUserWithdrawals(userID string) []*types.WithdrawalRequest {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var result []*types.WithdrawalRequest
	for _, w := range k.Withdrawals {
		if w.UserID == userID {
			result = append(result, w)
		}
	}
	return result
}

// GetStats returns settlement module statistics
func (k *Keeper) GetStats() map[string]interface{} {
	k.mu.RLock()
	defer k.mu.RUnlock()

	pendingDeposits := 0
	pendingWithdrawals := 0
	for _, d := range k.Deposits {
		if d.Status == "pending" {
			pendingDeposits++
		}
	}
	for _, w := range k.Withdrawals {
		if w.Status == "pending" {
			pendingWithdrawals++
		}
	}

	return map[string]interface{}{
		"total_settlements":       len(k.Settlements),
		"total_settled_usdc":      k.TotalSettledUSDC,
		"total_settled_usd":       float64(k.TotalSettledUSDC) / 1_000_000,
		"total_fees_collected":    k.TotalFeesCollected,
		"total_fees_usd":          float64(k.TotalFeesCollected) / 1_000_000,
		"total_deposited_usdc":    k.TotalDeposited,
		"total_deposited_usd":     float64(k.TotalDeposited) / 1_000_000,
		"total_withdrawn_usdc":    k.TotalWithdrawn,
		"total_withdrawn_usd":     float64(k.TotalWithdrawn) / 1_000_000,
		"active_escrows":          k.ActiveEscrows,
		"pending_deposits":        pendingDeposits,
		"pending_withdrawals":     pendingWithdrawals,
		"settlement_fee_bps":      k.Config.SettlementFeeBPS,
		"stripe_enabled":          k.Config.StripeEnabled,
		"crypto_enabled":          k.Config.CryptoEnabled,
		"bank_wire_enabled":       k.Config.BankWireEnabled,
		"noble_ibc_enabled":       k.Config.NobleIBCEnabled,
	}
}

// ============================================================
// HELPERS
// ============================================================

func (k *Keeper) isMethodEnabled(method types.PaymentMethod) bool {
	switch method {
	case types.PaymentMethodBankWire:
		return k.Config.BankWireEnabled
	case types.PaymentMethodStripe:
		return k.Config.StripeEnabled
	case types.PaymentMethodStablecoin:
		return k.Config.CryptoEnabled
	case types.PaymentMethodNobleIBC:
		return k.Config.NobleIBCEnabled
	case types.PaymentMethodCircleMint:
		return k.Config.CircleAPIEnabled
	}
	return false
}

func (k *Keeper) save() {
	os.MkdirAll(filepath.Dir(k.filePath), 0755)
	data, _ := json.MarshalIndent(k, "", "  ")
	os.WriteFile(k.filePath, data, 0644)
}
