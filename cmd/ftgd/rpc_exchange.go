package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/VelaFlux28/ftg-chain/x/exchange/keeper"
	"github.com/VelaFlux28/ftg-chain/x/exchange/types"
	kyckeeper "github.com/VelaFlux28/ftg-chain/x/kyc/keeper"
)

// ExchangeRPC handles all exchange-related HTTP endpoints
type ExchangeRPC struct {
	exchange *keeper.Keeper
	kyc      *kyckeeper.Keeper
	node     *FTGNode
}

// NewExchangeRPC creates the exchange RPC handler
func NewExchangeRPC(homeDir string, node *FTGNode) *ExchangeRPC {
	return &ExchangeRPC{
		exchange: keeper.NewKeeper(homeDir),
		kyc:      kyckeeper.NewKeeper(homeDir),
		node:     node,
	}
}

// RegisterRoutes registers all exchange endpoints on the given mux
func (e *ExchangeRPC) RegisterRoutes(mux *http.ServeMux) {
	// Wallet endpoints
	mux.HandleFunc("/exchange/wallet/create", e.handleCreateWallet)
	mux.HandleFunc("/exchange/wallet/get", e.handleGetWallet)
	mux.HandleFunc("/exchange/wallet/deposit", e.handleDeposit)
	mux.HandleFunc("/exchange/wallet/withdraw", e.handleWithdraw)

	// Order endpoints
	mux.HandleFunc("/exchange/order/place", e.handlePlaceOrder)
	mux.HandleFunc("/exchange/order/cancel", e.handleCancelOrder)
	mux.HandleFunc("/exchange/order/list", e.handleListOrders)

	// Market data endpoints
	mux.HandleFunc("/exchange/orderbook", e.handleOrderBook)
	mux.HandleFunc("/exchange/trades", e.handleRecentTrades)
	mux.HandleFunc("/exchange/pairs", e.handlePairs)
	mux.HandleFunc("/exchange/stats", e.handleExchangeStats)

	// KYC endpoints
	mux.HandleFunc("/exchange/kyc/submit", e.handleKYCSubmit)
	mux.HandleFunc("/exchange/kyc/approve", e.handleKYCApprove)
	mux.HandleFunc("/exchange/kyc/reject", e.handleKYCReject)
	mux.HandleFunc("/exchange/kyc/screen", e.handleKYCScreen)

	// Compliance endpoints
	mux.HandleFunc("/exchange/compliance/stats", e.handleComplianceStats)
	mux.HandleFunc("/exchange/compliance/flags", e.handleComplianceFlags)
	mux.HandleFunc("/exchange/compliance/resolve", e.handleComplianceResolve)
	mux.HandleFunc("/exchange/compliance/check-jurisdiction", e.handleCheckJurisdiction)
}

// ============================================================
// WALLET ENDPOINTS
// ============================================================

type CreateWalletRequest struct {
	UserID string `json:"user_id"`
	Label  string `json:"label"`
}

func (e *ExchangeRPC) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req CreateWalletRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	wallet, err := e.exchange.CreateWallet(req.UserID, req.Label)
	if err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, wallet)
}

func (e *ExchangeRPC) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	walletID := r.URL.Query().Get("wallet_id")
	userID := r.URL.Query().Get("user_id")

	var wallet *types.ExchangeWallet
	var err error

	if walletID != "" {
		wallet, err = e.exchange.GetWallet(walletID)
	} else if userID != "" {
		wallet, err = e.exchange.GetWalletByUser(userID)
	} else {
		e.errorResponse(w, "wallet_id or user_id required")
		return
	}

	if err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, wallet)
}

type DepositRequest struct {
	WalletID string `json:"wallet_id"`
	Denom    string `json:"denom"`
	Amount   int64  `json:"amount"`
}

func (e *ExchangeRPC) handleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req DepositRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	if err := e.exchange.DepositToWallet(req.WalletID, req.Denom, req.Amount); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]interface{}{
		"message":   "Deposit successful",
		"wallet_id": req.WalletID,
		"denom":     req.Denom,
		"amount":    req.Amount,
	})
}

type WithdrawRequest struct {
	WalletID string `json:"wallet_id"`
	Denom    string `json:"denom"`
	Amount   int64  `json:"amount"`
}

func (e *ExchangeRPC) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req WithdrawRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	if err := e.exchange.WithdrawFromWallet(req.WalletID, req.Denom, req.Amount); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]interface{}{
		"message":   "Withdrawal successful",
		"wallet_id": req.WalletID,
		"denom":     req.Denom,
		"amount":    req.Amount,
	})
}

// ============================================================
// ORDER ENDPOINTS
// ============================================================

type PlaceOrderRequest struct {
	WalletID  string `json:"wallet_id"`
	PairID    string `json:"pair_id"`
	Side      string `json:"side"`       // "buy" or "sell"
	OrderType string `json:"order_type"` // "limit" or "market"
	Price     int64  `json:"price"`      // in quote denom per 1M base denom
	Quantity  int64  `json:"quantity"`   // in base denom (uftg)
}

func (e *ExchangeRPC) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req PlaceOrderRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	// Validate side
	var side types.OrderSide
	switch req.Side {
	case "buy":
		side = types.OrderSideBuy
	case "sell":
		side = types.OrderSideSell
	default:
		e.errorResponse(w, "side must be 'buy' or 'sell'")
		return
	}

	// Validate order type
	var orderType types.OrderType
	switch req.OrderType {
	case "limit":
		orderType = types.OrderTypeLimit
	case "market":
		orderType = types.OrderTypeMarket
	default:
		e.errorResponse(w, "order_type must be 'limit' or 'market'")
		return
	}

	// Get current block height
	e.node.mu.RLock()
	height := e.node.State.Height
	e.node.mu.RUnlock()

	order, trades, err := e.exchange.PlaceOrder(req.WalletID, req.PairID, side, orderType, req.Price, req.Quantity, height)
	if err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]interface{}{
		"order":  order,
		"trades": trades,
		"message": fmt.Sprintf("Order %s placed. %d trade(s) executed.", order.ID, len(trades)),
	})
}

type CancelOrderRequest struct {
	OrderID  string `json:"order_id"`
	WalletID string `json:"wallet_id"`
}

func (e *ExchangeRPC) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req CancelOrderRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	if err := e.exchange.CancelOrder(req.OrderID, req.WalletID); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]interface{}{
		"message":  "Order cancelled",
		"order_id": req.OrderID,
	})
}

func (e *ExchangeRPC) handleListOrders(w http.ResponseWriter, r *http.Request) {
	walletID := r.URL.Query().Get("wallet_id")
	if walletID == "" {
		e.errorResponse(w, "wallet_id required")
		return
	}

	activeOnly := r.URL.Query().Get("active_only") == "true"
	orders := e.exchange.GetUserOrders(walletID, activeOnly)

	e.successResponse(w, map[string]interface{}{
		"orders":      orders,
		"total_count": len(orders),
	})
}

// ============================================================
// MARKET DATA ENDPOINTS
// ============================================================

func (e *ExchangeRPC) handleOrderBook(w http.ResponseWriter, r *http.Request) {
	pairID := r.URL.Query().Get("pair_id")
	if pairID == "" {
		pairID = "FTG-USDC"
	}

	depth := 20
	if d := r.URL.Query().Get("depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			depth = parsed
		}
	}

	book := e.exchange.GetOrderBook(pairID, depth)
	e.successResponse(w, book)
}

func (e *ExchangeRPC) handleRecentTrades(w http.ResponseWriter, r *http.Request) {
	pairID := r.URL.Query().Get("pair_id")
	if pairID == "" {
		pairID = "FTG-USDC"
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	trades := e.exchange.GetRecentTrades(pairID, limit)
	e.successResponse(w, map[string]interface{}{
		"trades":      trades,
		"total_count": len(trades),
	})
}

func (e *ExchangeRPC) handlePairs(w http.ResponseWriter, r *http.Request) {
	// Access pairs directly from the exchange stats
	stats := e.exchange.GetExchangeStats()
	e.successResponse(w, map[string]interface{}{
		"pairs": []map[string]interface{}{
			{
				"id":              "FTG-USDC",
				"base_symbol":     "FTG",
				"quote_symbol":    "USDC",
				"base_denom":      "uftg",
				"quote_denom":     "uusdc",
				"min_order_size":  1_000_000,
				"maker_fee_bps":   10,
				"taker_fee_bps":   25,
				"active":          true,
				"price_anchor":    "$5.00",
			},
		},
		"trading_enabled": stats["trading_enabled"],
	})
}

func (e *ExchangeRPC) handleExchangeStats(w http.ResponseWriter, r *http.Request) {
	stats := e.exchange.GetExchangeStats()
	e.successResponse(w, stats)
}

// ============================================================
// KYC ENDPOINTS
// ============================================================

type KYCSubmitRequest struct {
	UserID        string `json:"user_id"`
	WalletID      string `json:"wallet_id"`
	RequestedTier int    `json:"requested_tier"` // 1=basic, 2=full, 3=institutional
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Country       string `json:"country"`
	FullName      string `json:"full_name"`
	DateOfBirth   string `json:"date_of_birth"`
	IDType        string `json:"id_type"`
	IDNumber      string `json:"id_number"`
	CompanyName   string `json:"company_name"`
	CompanyRegNo  string `json:"company_reg_no"`
}

func (e *ExchangeRPC) handleKYCSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req KYCSubmitRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	app := &types.KYCApplication{
		UserID:        req.UserID,
		WalletID:      req.WalletID,
		RequestedTier: types.KYCTier(req.RequestedTier),
		Email:         req.Email,
		Phone:         req.Phone,
		Country:       req.Country,
		FullName:      req.FullName,
		DateOfBirth:   req.DateOfBirth,
		IDType:        req.IDType,
		IDNumber:      req.IDNumber,
		CompanyName:   req.CompanyName,
		CompanyRegNo:  req.CompanyRegNo,
	}

	result, err := e.exchange.SubmitKYC(app)
	if err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	// Also run the KYC module screening
	if req.FullName != "" && req.Country != "" {
		e.kyc.ScreenUser(req.UserID, req.FullName, req.Country, req.IDNumber)
	}

	e.successResponse(w, result)
}

type KYCApproveRequest struct {
	ApplicationID string `json:"application_id"`
}

func (e *ExchangeRPC) handleKYCApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req KYCApproveRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	if err := e.exchange.ApproveKYC(req.ApplicationID); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]string{
		"message":        "KYC application approved",
		"application_id": req.ApplicationID,
	})
}

type KYCRejectRequest struct {
	ApplicationID string `json:"application_id"`
	Reason        string `json:"reason"`
}

func (e *ExchangeRPC) handleKYCReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req KYCRejectRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	if err := e.exchange.RejectKYC(req.ApplicationID, req.Reason); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]string{
		"message":        "KYC application rejected",
		"application_id": req.ApplicationID,
		"reason":         req.Reason,
	})
}

type ScreenRequest struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Country  string `json:"country"`
	IDNumber string `json:"id_number"`
}

func (e *ExchangeRPC) handleKYCScreen(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ScreenRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	result, err := e.kyc.ScreenUser(req.UserID, req.FullName, req.Country, req.IDNumber)
	if err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, result)
}

// ============================================================
// COMPLIANCE ENDPOINTS
// ============================================================

func (e *ExchangeRPC) handleComplianceStats(w http.ResponseWriter, r *http.Request) {
	stats := e.kyc.GetComplianceStats()
	e.successResponse(w, stats)
}

func (e *ExchangeRPC) handleComplianceFlags(w http.ResponseWriter, r *http.Request) {
	flags := e.kyc.GetPendingFlags()
	e.successResponse(w, map[string]interface{}{
		"pending_flags": flags,
		"total_count":   len(flags),
	})
}

type ResolveFlagRequest struct {
	FlagID     string `json:"flag_id"`
	Resolution string `json:"resolution"` // cleared, escalated, frozen
}

func (e *ExchangeRPC) handleComplianceResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req ResolveFlagRequest
	if err := e.readJSON(r, &req); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	if err := e.kyc.ResolveFlag(req.FlagID, req.Resolution); err != nil {
		e.errorResponse(w, err.Error())
		return
	}

	e.successResponse(w, map[string]string{
		"message":    "Flag resolved",
		"flag_id":    req.FlagID,
		"resolution": req.Resolution,
	})
}

func (e *ExchangeRPC) handleCheckJurisdiction(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" {
		e.errorResponse(w, "country query parameter required (ISO 3166-1 alpha-2)")
		return
	}

	blocked, reason := e.kyc.IsJurisdictionBlocked(country)
	e.successResponse(w, map[string]interface{}{
		"country_code": country,
		"blocked":      blocked,
		"reason":       reason,
	})
}

// ============================================================
// HELPERS
// ============================================================

func (e *ExchangeRPC) readJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func (e *ExchangeRPC) successResponse(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

func (e *ExchangeRPC) errorResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}
