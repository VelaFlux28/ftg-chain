package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChainState holds the full application state
type ChainState struct {
	mu sync.RWMutex

	// Token supply
	TotalMinted  int64 `json:"total_minted"`  // Total FTG minted (in uftg = micro-FTG)
	TotalBurned  int64 `json:"total_burned"`  // Total FTG burned
	TotalSupply  int64 `json:"total_supply"`  // Current circulating supply
	TotalMWh     int64 `json:"total_mwh"`     // Total MWh backing

	// Treasury balances (in uftg)
	TreasuryMinting   int64 `json:"treasury_minting"`
	TreasuryCommitted int64 `json:"treasury_committed"`
	TreasuryLiquidity int64 `json:"treasury_liquidity"`

	// Certificates
	Certificates []Certificate `json:"certificates"`

	// Burn events
	BurnEvents []BurnEvent `json:"burn_events"`

	// Transactions
	Transactions []Transaction `json:"transactions"`

	// File path for persistence
	filePath string
}

// Certificate represents a registered energy certificate
type Certificate struct {
	ID             string    `json:"id"`
	ExternalID     string    `json:"external_id"`
	Provider       string    `json:"provider"`
	MWhAmount      int64     `json:"mwh_amount"`
	TokensMinted   int64     `json:"tokens_minted"` // in uftg
	IPFSHash       string    `json:"ipfs_hash"`
	ProvenanceType string    `json:"provenance_type"`
	SourceType     string    `json:"source_type"`
	VintageYear    int       `json:"vintage_year"`
	MintedAt       time.Time `json:"minted_at"`
	MintedInBlock  int64     `json:"minted_in_block"`
	Minter         string    `json:"minter"`
}

// BurnEvent represents a token burn
type BurnEvent struct {
	ID        string    `json:"id"`
	Amount    int64     `json:"amount"` // in uftg
	BurnType  string    `json:"burn_type"` // merchant_protocol, manual, redemption
	Burner    string    `json:"burner"`
	Reason    string    `json:"reason"`
	BurnedAt  time.Time `json:"burned_at"`
	BlockHeight int64   `json:"block_height"`
}

// Transaction represents a processed transaction
type Transaction struct {
	Hash      string    `json:"hash"`
	Type      string    `json:"type"`
	Height    int64     `json:"height"`
	Timestamp time.Time `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
	Status    string    `json:"status"` // success, failed
}

// NewChainState creates or loads chain state
func NewChainState(homeDir string) *ChainState {
	statePath := filepath.Join(homeDir, "data", "chain_state.json")

	cs := &ChainState{
		filePath:     statePath,
		Certificates: make([]Certificate, 0),
		BurnEvents:   make([]BurnEvent, 0),
		Transactions: make([]Transaction, 0),
	}

	// Try to load existing state
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, cs)
	}

	return cs
}

// MintFromCertificate processes a mint transaction
func (cs *ChainState) MintFromCertificate(cert Certificate, blockHeight int64) (*Transaction, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Validate: check for duplicate certificate
	for _, existing := range cs.Certificates {
		if existing.ExternalID == cert.ExternalID {
			return nil, fmt.Errorf("certificate %s already registered (double-mint prevention)", cert.ExternalID)
		}
	}

	// Calculate tokens: 1 MWh = 100 FTG = 100,000,000 uftg
	tokensUFTG := cert.MWhAmount * 100 * 1_000_000 // 100 FTG per MWh, 1M uftg per FTG

	// Update state
	cert.TokensMinted = tokensUFTG
	cert.MintedAt = time.Now().UTC()
	cert.MintedInBlock = blockHeight

	cs.Certificates = append(cs.Certificates, cert)
	cs.TotalMinted += tokensUFTG
	cs.TotalSupply += tokensUFTG
	cs.TotalMWh += cert.MWhAmount
	cs.TreasuryMinting += tokensUFTG

	// Create transaction record
	txData, _ := json.Marshal(map[string]interface{}{
		"certificate_id": cert.ExternalID,
		"mwh_amount":     cert.MWhAmount,
		"tokens_minted":  tokensUFTG,
		"provider":       cert.Provider,
	})

	tx := &Transaction{
		Hash:      fmt.Sprintf("MINT_%s_%d", cert.ExternalID, blockHeight),
		Type:      "energymint/MsgMintFromCertificate",
		Height:    blockHeight,
		Timestamp: time.Now().UTC(),
		Data:      txData,
		Status:    "success",
	}

	cs.Transactions = append(cs.Transactions, *tx)

	// Persist state
	cs.save()

	return tx, nil
}

// BurnTokens processes a burn transaction
func (cs *ChainState) BurnTokens(event BurnEvent, blockHeight int64) (*Transaction, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Validate: check sufficient supply
	if event.Amount > cs.TotalSupply {
		return nil, fmt.Errorf("insufficient supply: want to burn %d but only %d exists", event.Amount, cs.TotalSupply)
	}

	event.BurnedAt = time.Now().UTC()
	event.BlockHeight = blockHeight

	cs.BurnEvents = append(cs.BurnEvents, event)
	cs.TotalBurned += event.Amount
	cs.TotalSupply -= event.Amount

	// Create transaction record
	txData, _ := json.Marshal(map[string]interface{}{
		"amount":    event.Amount,
		"burn_type": event.BurnType,
		"reason":    event.Reason,
	})

	tx := &Transaction{
		Hash:      fmt.Sprintf("BURN_%s_%d", event.ID, blockHeight),
		Type:      "ftgburn/MsgBurn",
		Height:    blockHeight,
		Timestamp: time.Now().UTC(),
		Data:      txData,
		Status:    "success",
	}

	cs.Transactions = append(cs.Transactions, *tx)

	// Persist state
	cs.save()

	return tx, nil
}

// TransferFromMintingToCommitted moves tokens for OTC deals
func (cs *ChainState) TransferFromMintingToCommitted(amount int64, buyerRef string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if amount > cs.TreasuryMinting {
		return fmt.Errorf("insufficient minting balance: want %d but only %d available", amount, cs.TreasuryMinting)
	}

	cs.TreasuryMinting -= amount
	cs.TreasuryCommitted += amount
	cs.save()
	return nil
}

// TransferFromMintingToLiquidity moves tokens to DEX
func (cs *ChainState) TransferFromMintingToLiquidity(amount int64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if amount > cs.TreasuryMinting {
		return fmt.Errorf("insufficient minting balance: want %d but only %d available", amount, cs.TreasuryMinting)
	}

	cs.TreasuryMinting -= amount
	cs.TreasuryLiquidity += amount
	cs.save()
	return nil
}

// GetStats returns current chain statistics
func (cs *ChainState) GetStats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return map[string]interface{}{
		"total_minted_uftg":    cs.TotalMinted,
		"total_minted_ftg":     float64(cs.TotalMinted) / 1_000_000,
		"total_burned_uftg":    cs.TotalBurned,
		"total_burned_ftg":     float64(cs.TotalBurned) / 1_000_000,
		"total_supply_uftg":    cs.TotalSupply,
		"total_supply_ftg":     float64(cs.TotalSupply) / 1_000_000,
		"total_mwh_backed":     cs.TotalMWh,
		"backing_ratio":        "100%",
		"price_anchor":         "$5.00/FTG",
		"certificates_count":   len(cs.Certificates),
		"burn_events_count":    len(cs.BurnEvents),
		"transactions_count":   len(cs.Transactions),
		"treasury_minting_ftg":   float64(cs.TreasuryMinting) / 1_000_000,
		"treasury_committed_ftg": float64(cs.TreasuryCommitted) / 1_000_000,
		"treasury_liquidity_ftg": float64(cs.TreasuryLiquidity) / 1_000_000,
	}
}

// save persists chain state to disk
func (cs *ChainState) save() {
	os.MkdirAll(filepath.Dir(cs.filePath), 0755)
	data, _ := json.MarshalIndent(cs, "", "  ")
	os.WriteFile(cs.filePath, data, 0644)
}
