package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// RPCServer handles all JSON-RPC endpoints for the FTG chain
type RPCServer struct {
	node          *FTGNode
	chainState    *ChainState
	httpServer    *http.Server
	ExchangeRPC   *ExchangeRPC
	SettlementRPC *SettlementRPC
}

// NewRPCServer creates the RPC server with all endpoints
func NewRPCServer(node *FTGNode, chainState *ChainState) *RPCServer {
	return &RPCServer{
		node:       node,
		chainState: chainState,
	}
}

// Start begins listening on port 26657
func (s *RPCServer) Start() error {
	mux := http.NewServeMux()

	// Core CometBFT-compatible endpoints
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/block", s.handleBlock)
	mux.HandleFunc("/genesis", s.handleGenesis)
	mux.HandleFunc("/net_info", s.handleNetInfo)

	// Transaction endpoints
	mux.HandleFunc("/broadcast_tx_sync", s.handleBroadcastTx)
	mux.HandleFunc("/broadcast_tx_commit", s.handleBroadcastTx)

	// FTG-specific endpoints
	mux.HandleFunc("/ftg/mint", s.handleMint)
	mux.HandleFunc("/ftg/burn", s.handleBurn)
	mux.HandleFunc("/ftg/stats", s.handleStats)
	mux.HandleFunc("/ftg/certificates", s.handleCertificates)
	mux.HandleFunc("/ftg/treasury", s.handleTreasury)
	mux.HandleFunc("/ftg/treasury/commit", s.handleTreasuryCommit)
	mux.HandleFunc("/ftg/treasury/to-liquidity", s.handleTreasuryToLiquidity)
	mux.HandleFunc("/ftg/transactions", s.handleTransactions)

	// ABCI query endpoint
	mux.HandleFunc("/abci_query", s.handleABCIQuery)

	// Exchange module endpoints
	if s.ExchangeRPC != nil {
		s.ExchangeRPC.RegisterRoutes(mux)
	}

	// Settlement bridge endpoints
	if s.SettlementRPC != nil {
		s.SettlementRPC.RegisterRoutes(mux)
	}

	listener, err := net.Listen("tcp", "0.0.0.0:26657")
	if err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	s.httpServer = &http.Server{Handler: mux}
	fmt.Printf("  RPC server listening on http://0.0.0.0:26657\n")
	return s.httpServer.Serve(listener)
}

// handleStatus returns node status
func (s *RPCServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.node.mu.RLock()
	defer s.node.mu.RUnlock()

	status := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"node_info": map[string]interface{}{
				"protocol_version": map[string]string{
					"p2p": "8", "block": "11", "app": "0",
				},
				"id":      "4cc50879b244c02262b83ddf835efd5d4a6df34c",
				"network": s.node.ChainID,
				"moniker": s.node.Moniker,
				"version": "0.38.0",
			},
			"sync_info": map[string]interface{}{
				"latest_block_hash":   s.node.State.AppHash,
				"latest_block_height": fmt.Sprintf("%d", s.node.State.Height),
				"latest_block_time":   s.node.State.Timestamp.Format(time.RFC3339Nano),
				"catching_up":         false,
			},
			"validator_info": map[string]interface{}{
				"address":      "4CC50879B244C02262B83DDF835EFD5D4A6DF34C",
				"voting_power": "1000000",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleHealth returns health check
func (s *RPCServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  map[string]interface{}{},
	})
}

// handleBlock returns current block info
func (s *RPCServer) handleBlock(w http.ResponseWriter, r *http.Request) {
	s.node.mu.RLock()
	defer s.node.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"block": map[string]interface{}{
				"header": map[string]interface{}{
					"chain_id": s.node.ChainID,
					"height":   fmt.Sprintf("%d", s.node.State.Height),
					"time":     s.node.State.Timestamp.Format(time.RFC3339Nano),
					"app_hash": s.node.State.AppHash,
				},
			},
		},
	})
}

// handleGenesis returns the genesis file
func (s *RPCServer) handleGenesis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"genesis": map[string]interface{}{
				"chain_id":     s.node.ChainID,
				"genesis_time": "2026-05-10T00:00:00Z",
			},
		},
	})
}

// handleNetInfo returns network info
func (s *RPCServer) handleNetInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"listening": true,
			"listeners": []string{"Listener(@tcp://0.0.0.0:26656)"},
			"n_peers":   "0",
			"peers":     []interface{}{},
		},
	})
}

// handleBroadcastTx handles generic transaction broadcast
func (s *RPCServer) handleBroadcastTx(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"code": 0,
			"hash": fmt.Sprintf("TX_%d", time.Now().UnixNano()),
			"log":  "tx accepted",
		},
	})
}

// MintRequest represents a mint transaction request
type MintRequest struct {
	CertificateID  string `json:"certificate_id"`
	Provider       string `json:"provider"`
	MWhAmount      int64  `json:"mwh_amount"`
	IPFSHash       string `json:"ipfs_hash"`
	ProvenanceType string `json:"provenance_type"`
	SourceType     string `json:"source_type"`
	VintageYear    int    `json:"vintage_year"`
	Minter         string `json:"minter"`
}

// handleMint processes a mint transaction
func (s *RPCServer) handleMint(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, "failed to read request body")
		return
	}

	var req MintRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.errorResponse(w, "invalid JSON: "+err.Error())
		return
	}

	// Validate required fields
	if req.CertificateID == "" || req.MWhAmount <= 0 {
		s.errorResponse(w, "certificate_id and mwh_amount > 0 are required")
		return
	}

	// Get current block height
	s.node.mu.RLock()
	height := s.node.State.Height
	s.node.mu.RUnlock()

	// Create certificate and mint
	cert := Certificate{
		ID:             fmt.Sprintf("cert_%d_%s", height, req.CertificateID),
		ExternalID:     req.CertificateID,
		Provider:       req.Provider,
		MWhAmount:      req.MWhAmount,
		IPFSHash:       req.IPFSHash,
		ProvenanceType: req.ProvenanceType,
		SourceType:     req.SourceType,
		VintageYear:    req.VintageYear,
		Minter:         req.Minter,
	}

	tx, err := s.chainState.MintFromCertificate(cert, height)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	// Calculate human-readable amounts
	ftgMinted := float64(cert.MWhAmount) * 100 // 100 FTG per MWh

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"tx_hash":        tx.Hash,
			"block_height":   height,
			"certificate_id": req.CertificateID,
			"mwh_amount":     req.MWhAmount,
			"ftg_minted":     ftgMinted,
			"uftg_minted":    cert.TokensMinted,
			"provider":       req.Provider,
			"ipfs_hash":      req.IPFSHash,
			"timestamp":      tx.Timestamp.Format(time.RFC3339),
			"message":        fmt.Sprintf("Successfully minted %.0f FTG from %d MWh certificate", ftgMinted, req.MWhAmount),
		},
	})
}

// BurnRequest represents a burn transaction request
type BurnRequest struct {
	Amount   int64  `json:"amount"` // in uftg
	BurnType string `json:"burn_type"`
	Burner   string `json:"burner"`
	Reason   string `json:"reason"`
}

// handleBurn processes a burn transaction
func (s *RPCServer) handleBurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.errorResponse(w, "failed to read request body")
		return
	}

	var req BurnRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.errorResponse(w, "invalid JSON: "+err.Error())
		return
	}

	s.node.mu.RLock()
	height := s.node.State.Height
	s.node.mu.RUnlock()

	event := BurnEvent{
		ID:       fmt.Sprintf("burn_%d", time.Now().UnixNano()),
		Amount:   req.Amount,
		BurnType: req.BurnType,
		Burner:   req.Burner,
		Reason:   req.Reason,
	}

	tx, err := s.chainState.BurnTokens(event, height)
	if err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"tx_hash":      tx.Hash,
			"block_height": height,
			"amount_uftg":  req.Amount,
			"amount_ftg":   float64(req.Amount) / 1_000_000,
			"burn_type":    req.BurnType,
			"timestamp":    tx.Timestamp.Format(time.RFC3339),
		},
	})
}

// handleStats returns chain statistics
func (s *RPCServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.chainState.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  stats,
	})
}

// handleCertificates returns all registered certificates
func (s *RPCServer) handleCertificates(w http.ResponseWriter, r *http.Request) {
	s.chainState.mu.RLock()
	defer s.chainState.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"certificates": s.chainState.Certificates,
			"total_count":  len(s.chainState.Certificates),
		},
	})
}

// handleTreasury returns treasury wallet balances
func (s *RPCServer) handleTreasury(w http.ResponseWriter, r *http.Request) {
	s.chainState.mu.RLock()
	defer s.chainState.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"minting_wallet": map[string]interface{}{
				"balance_uftg": s.chainState.TreasuryMinting,
				"balance_ftg":  float64(s.chainState.TreasuryMinting) / 1_000_000,
				"description":  "Newly minted tokens awaiting allocation",
			},
			"committed_wallet": map[string]interface{}{
				"balance_uftg": s.chainState.TreasuryCommitted,
				"balance_ftg":  float64(s.chainState.TreasuryCommitted) / 1_000_000,
				"description":  "Reserved for OTC buyers — off public market",
			},
			"liquidity_wallet": map[string]interface{}{
				"balance_uftg": s.chainState.TreasuryLiquidity,
				"balance_ftg":  float64(s.chainState.TreasuryLiquidity) / 1_000_000,
				"description":  "Available for public trading on DEX",
			},
			"total_treasury_ftg": float64(s.chainState.TreasuryMinting+s.chainState.TreasuryCommitted+s.chainState.TreasuryLiquidity) / 1_000_000,
		},
	})
}

// TreasuryCommitRequest represents a commit request
type TreasuryCommitRequest struct {
	Amount   int64  `json:"amount"` // in uftg
	BuyerRef string `json:"buyer_ref"`
}

// handleTreasuryCommit moves tokens from Minting → Committed
func (s *RPCServer) handleTreasuryCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req TreasuryCommitRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.errorResponse(w, "invalid JSON")
		return
	}

	if err := s.chainState.TransferFromMintingToCommitted(req.Amount, req.BuyerRef); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"amount_ftg": float64(req.Amount) / 1_000_000,
			"buyer_ref":  req.BuyerRef,
			"message":    "Tokens moved from Minting → Committed. Off public market.",
		},
	})
}

// TreasuryLiquidityRequest represents a liquidity transfer request
type TreasuryLiquidityRequest struct {
	Amount int64 `json:"amount"` // in uftg
}

// handleTreasuryToLiquidity moves tokens from Minting → Liquidity
func (s *RPCServer) handleTreasuryToLiquidity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req TreasuryLiquidityRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.errorResponse(w, "invalid JSON")
		return
	}

	if err := s.chainState.TransferFromMintingToLiquidity(req.Amount); err != nil {
		s.errorResponse(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"amount_ftg": float64(req.Amount) / 1_000_000,
			"message":    "Tokens moved from Minting → Liquidity. Available on DEX.",
		},
	})
}

// handleTransactions returns transaction history
func (s *RPCServer) handleTransactions(w http.ResponseWriter, r *http.Request) {
	s.chainState.mu.RLock()
	defer s.chainState.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"transactions": s.chainState.Transactions,
			"total_count":  len(s.chainState.Transactions),
		},
	})
}

// handleABCIQuery handles ABCI query requests
func (s *RPCServer) handleABCIQuery(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	var responseValue interface{}
	switch path {
	case "/ftgchain.energymint.v1.Query/TotalMinted":
		responseValue = map[string]interface{}{
			"total_minted_ftg": float64(s.chainState.TotalMinted) / 1_000_000,
			"total_mwh":        s.chainState.TotalMWh,
		}
	case "/ftgchain.burn.v1.Query/BurnStats":
		responseValue = map[string]interface{}{
			"total_burned_ftg": float64(s.chainState.TotalBurned) / 1_000_000,
			"total_events":     len(s.chainState.BurnEvents),
		}
	default:
		responseValue = map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"response": map[string]interface{}{
				"code":  0,
				"value": responseValue,
			},
		},
	})
}

// errorResponse sends an error JSON response
func (s *RPCServer) errorResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}
