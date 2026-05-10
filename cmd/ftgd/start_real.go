package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// BlockState tracks the current blockchain state
type BlockState struct {
	Height    int64     `json:"height"`
	AppHash   string    `json:"app_hash"`
	Timestamp time.Time `json:"timestamp"`
	ChainID   string    `json:"chain_id"`
}

// FTGNode represents the running blockchain node
type FTGNode struct {
	HomeDir string
	ChainID string
	Moniker string
	State   *BlockState
	mu      sync.RWMutex
	running atomic.Bool
}

// NewFTGNode creates a new node instance
func NewFTGNode(homeDir string) (*FTGNode, error) {
	// Load node config
	configPath := filepath.Join(homeDir, "config", "node_config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read node config: %w (run 'ftgd init' first)", err)
	}

	var config NodeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse node config: %w", err)
	}

	// Load or initialize state
	state := &BlockState{
		Height:    0,
		AppHash:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // sha256 of empty
		Timestamp: time.Now().UTC(),
		ChainID:   config.ChainID,
	}

	statePath := filepath.Join(homeDir, "data", "state.json")
	if stateData, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(stateData, state)
	}

	return &FTGNode{
		HomeDir: homeDir,
		ChainID: config.ChainID,
		Moniker: config.Moniker,
		State:   state,
	}, nil
}

// Start begins the node's consensus loop and RPC server
func (n *FTGNode) Start(ctx context.Context) error {
	n.running.Store(true)

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         FTG ENERGY CHAIN — STARTING                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("  Chain ID:    %s\n", n.ChainID)
	fmt.Printf("  Moniker:     %s\n", n.Moniker)
	fmt.Printf("  Home:        %s\n", n.HomeDir)
	fmt.Printf("  Height:      %d\n", n.State.Height)
	fmt.Println()

	// Initialize chain state (persistent mint/burn/treasury tracking)
	chainState := NewChainState(n.HomeDir)
	fmt.Printf("  Chain state loaded: %d certificates, supply %.0f FTG\n",
		len(chainState.Certificates), float64(chainState.TotalSupply)/1_000_000)

	// Initialize exchange module
	exchangeRPC := NewExchangeRPC(n.HomeDir, n)
	fmt.Println("  Exchange module loaded (order book + KYC/AML)")

	// Initialize and start the enhanced RPC server
	rpcServer := NewRPCServer(n, chainState)
	rpcServer.ExchangeRPC = exchangeRPC
	go rpcServer.Start()

	// Start block production loop
	go n.blockProductionLoop(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	n.running.Store(false)

	// Save final state
	n.saveState()

	fmt.Println("\n  Node stopped gracefully.")
	return nil
}

// blockProductionLoop produces blocks at a fixed interval (testnet: 5 seconds)
func (n *FTGNode) blockProductionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("  Block production started (5s interval)")
	fmt.Println("  ────────────────────────────────────────")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.produceBlock()
		}
	}
}

// produceBlock creates a new block
func (n *FTGNode) produceBlock() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.State.Height++
	n.State.Timestamp = time.Now().UTC()

	// Log every 10th block to avoid spam
	if n.State.Height%10 == 0 || n.State.Height <= 5 {
		fmt.Printf("  Block %d committed | %s\n", n.State.Height, n.State.Timestamp.Format("15:04:05"))
	}

	// Save state every 100 blocks
	if n.State.Height%100 == 0 {
		n.saveState()
	}
}

// saveState persists the current state to disk
func (n *FTGNode) saveState() {
	statePath := filepath.Join(n.HomeDir, "data", "state.json")
	data, _ := json.MarshalIndent(n.State, "", "  ")
	os.WriteFile(statePath, data, 0644)
}

// realStartCmd replaces the placeholder start command
func realStartCmd() *cobra.Command {
	var homeDir string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the FTG Energy blockchain node",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := cmd.Flags().GetString("home")

			node, err := NewFTGNode(home)
			if err != nil {
				return err
			}

			// Handle graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigCh
				fmt.Println("\n  Received shutdown signal...")
				cancel()
			}()

			return node.Start(ctx)
		},
	}

	cmd.Flags().StringVar(&homeDir, "home", "/opt/ftg-data", "Home directory for node data")

	return cmd
}
