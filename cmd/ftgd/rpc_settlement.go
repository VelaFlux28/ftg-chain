package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/VelaFlux28/ftg-chain/x/settlement/keeper"
	"github.com/VelaFlux28/ftg-chain/x/settlement/types"
)

// SettlementRPC handles all settlement-related HTTP endpoints
type SettlementRPC struct {
	settlement *keeper.Keeper
	node       *FTGNode
}

// NewSettlementRPC creates the settlement RPC handler
func NewSettlementRPC(homeDir string, node *FTGNode) *SettlementRPC {
	return &SettlementRPC{
		settlement: keeper.NewKeeper(homeDir),
		node:       node,
	}
}

// RegisterRoutes registers all settlement endpoints on the given mux
func (s *SettlementRPC) RegisterRoutes(mux *http.ServeMux) {
	// Deposit endpoints
	mux.HandleFunc("/settlement/deposit/create", s.handleCreateDeposit)
	mux.HandleFunc("/settlement/deposit/confirm", s.handleConfirmDeposit)
	mux.HandleFunc("/settlement/deposit/get", s.handleGetDeposit)
	mux.HandleFunc("/settlement/deposit/list", s.handleListDeposits)

	// Withdrawal endpoints
	mux.HandleFunc("/settlement/withdrawal/create", s.handleCreateWithdrawal)
	mux.HandleFunc("/settlement/withdrawal/process", s.handleProcessWithdrawal)
	mux.HandleFunc("/settlement/withdrawal/list", s.handleListWithdrawals)

	// Escrow endpoints
	mux.HandleFunc("/settlement/escrow/create", s.handleCreateEscrow)
	mux.HandleFunc("/settlement/escrow/fund", s.handleFundEscrow)
	mux.HandleFunc("/settlement/escrow/execute", s.handleExecuteSettlement)
	mux.HandleFunc("/settlement/escrow/refund", s.handleRefundEscrow)
	mux.HandleFunc("/settlement/escrow/get", s.handleGetEscrow)

	// Query endpoints
	mux.HandleFunc("/settlement/history", s.handleSettlementHistory)
	mux.HandleFunc("/settlement/stats", s.handleSettlementStats)
}

// ============================================================
// DEPOSIT ENDPOINTS
// ============================================================

type CreateDepositRequest struct {
	WalletID      string `json:"wallet_id"`
	UserID        string `json:"user_id"`
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method"`
}

func (s *SettlementRPC) handleCreateDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req CreateDepositRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	intent, err := s.settlement.CreateDepositIntent(req.WalletID, req.UserID, req.Amount, types.PaymentMethod(req.PaymentMethod))
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, intent)
}

type ConfirmDepositRequest struct {
	DepositID  string `json:"deposit_id"`
	PaymentRef string `json:"payment_ref"`
}

func (s *SettlementRPC) handleConfirmDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ConfirmDepositRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	intent, err := s.settlement.ConfirmDeposit(req.DepositID, req.PaymentRef)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, intent)
}

func (s *SettlementRPC) handleGetDeposit(w http.ResponseWriter, r *http.Request) {
	depositID := r.URL.Query().Get("deposit_id")
	if depositID == "" {
		s.errorResponse(w, "deposit_id required")
		return
	}

	dep, err := s.settlement.GetDeposit(depositID)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, dep)
}

func (s *SettlementRPC) handleListDeposits(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		s.errorResponse(w, "user_id required")
		return
	}

	deposits := s.settlement.GetUserDeposits(userID)
	s.successResponse(w, map[string]interface{}{
		"deposits":    deposits,
		"total_count": len(deposits),
	})
}

// ============================================================
// WITHDRAWAL ENDPOINTS
// ============================================================

type CreateWithdrawalRequest struct {
	WalletID       string `json:"wallet_id"`
	UserID         string `json:"user_id"`
	Amount         int64  `json:"amount"`
	PaymentMethod  string `json:"payment_method"`
	DestinationRef string `json:"destination_ref"`
}

func (s *SettlementRPC) handleCreateWithdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req CreateWithdrawalRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	withdrawal, err := s.settlement.CreateWithdrawalRequest(req.WalletID, req.UserID, req.Amount, types.PaymentMethod(req.PaymentMethod), req.DestinationRef)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, withdrawal)
}

type ProcessWithdrawalRequest struct {
	WithdrawalID string `json:"withdrawal_id"`
}

func (s *SettlementRPC) handleProcessWithdrawal(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ProcessWithdrawalRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	withdrawal, err := s.settlement.ProcessWithdrawal(req.WithdrawalID)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, withdrawal)
}

func (s *SettlementRPC) handleListWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		s.errorResponse(w, "user_id required")
		return
	}

	withdrawals := s.settlement.GetUserWithdrawals(userID)
	s.successResponse(w, map[string]interface{}{
		"withdrawals": withdrawals,
		"total_count": len(withdrawals),
	})
}

// ============================================================
// ESCROW ENDPOINTS
// ============================================================

type CreateEscrowRequest struct {
	TradeID        string `json:"trade_id"`
	BuyerWalletID  string `json:"buyer_wallet_id"`
	SellerWalletID string `json:"seller_wallet_id"`
	AmountUSDC     int64  `json:"amount_usdc"`
	PaymentMethod  string `json:"payment_method"`
}

func (s *SettlementRPC) handleCreateEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req CreateEscrowRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	escrow, err := s.settlement.CreateEscrow(req.TradeID, req.BuyerWalletID, req.SellerWalletID, req.AmountUSDC, types.PaymentMethod(req.PaymentMethod))
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, escrow)
}

type FundEscrowRequest struct {
	EscrowID   string `json:"escrow_id"`
	PaymentRef string `json:"payment_ref"`
}

func (s *SettlementRPC) handleFundEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req FundEscrowRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	escrow, err := s.settlement.FundEscrow(req.EscrowID, req.PaymentRef)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, escrow)
}

type ExecuteSettlementRequest struct {
	EscrowID  string `json:"escrow_id"`
	FTGAmount int64  `json:"ftg_amount"`
}

func (s *SettlementRPC) handleExecuteSettlement(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteSettlementRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.node.mu.RLock()
	height := s.node.State.Height
	s.node.mu.RUnlock()

	record, err := s.settlement.ExecuteSettlement(req.EscrowID, req.FTGAmount, height)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, record)
}

type RefundEscrowRequest struct {
	EscrowID string `json:"escrow_id"`
	Reason   string `json:"reason"`
}

func (s *SettlementRPC) handleRefundEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req RefundEscrowRequest
	if err := s.readJSON(r, &req); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	if err := s.settlement.RefundEscrow(req.EscrowID, req.Reason); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, map[string]string{
		"message":   "Escrow refunded",
		"escrow_id": req.EscrowID,
	})
}

func (s *SettlementRPC) handleGetEscrow(w http.ResponseWriter, r *http.Request) {
	escrowID := r.URL.Query().Get("escrow_id")
	if escrowID == "" {
		s.errorResponse(w, "escrow_id required")
		return
	}

	escrow, err := s.settlement.GetEscrow(escrowID)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	s.successResponse(w, escrow)
}

// ============================================================
// QUERY ENDPOINTS
// ============================================================

func (s *SettlementRPC) handleSettlementHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	settlements := s.settlement.GetSettlements(limit)
	s.successResponse(w, map[string]interface{}{
		"settlements": settlements,
		"total_count": len(settlements),
	})
}

func (s *SettlementRPC) handleSettlementStats(w http.ResponseWriter, r *http.Request) {
	stats := s.settlement.GetStats()
	s.successResponse(w, stats)
}

// ============================================================
// HELPERS
// ============================================================

func (s *SettlementRPC) readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func (s *SettlementRPC) successResponse(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

func (s *SettlementRPC) errorResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}
