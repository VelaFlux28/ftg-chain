package main

import (
	"fmt"
	"os"
)

// ftgd is the FTG Energy blockchain daemon
// This is the binary that runs as a validator node on the FTG Energy chain
//
// Chain: ftg-energy-1
// Token: FTG (1 FTG = 10 kWh stored energy)
// Consensus: CometBFT (Tendermint)
// Framework: Cosmos SDK v0.50
//
// Custom Modules:
//   - x/energymint: Mint FTG tokens from verified energy certificates (RECs)
//   - x/ftgburn: Automatic Merchant Protocol (energy sales → buy → burn)
//   - x/provenance: Immutable certificate registry with IPFS integration
//   - x/treasury: Segregated wallet management (Minting, Committed, Liquidity)
//
// Patents:
//   - US 11,962,710 (Energy Block Chain)
//   - 17/863,165 (Gateway/Toll Booth — allowed April 2026)
//
// Regulatory: SEC Release 33-11412 — Digital Commodity classification

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
