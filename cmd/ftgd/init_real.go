package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// NodeConfig represents the CometBFT node configuration
type NodeConfig struct {
	Moniker    string `json:"moniker"`
	ChainID    string `json:"chain_id"`
	NodeID     string `json:"node_id"`
	ListenAddr string `json:"listen_addr"`
	RPCAddr    string `json:"rpc_addr"`
	P2PAddr    string `json:"p2p_addr"`
}

// ValidatorKey represents the validator's consensus key
type ValidatorKey struct {
	Address string `json:"address"`
	PubKey  struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"pub_key"`
	PrivKey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"priv_key"`
}

// realInitCmd creates the actual node initialization
func realInitCmd() *cobra.Command {
	var homeDir string
	var testnetChainID string

	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize a new FTG Energy node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moniker := args[0]
			home, _ := cmd.Flags().GetString("home")
			cid, _ := cmd.Flags().GetString("chain-id")

			fmt.Printf("Initializing FTG Energy node: %s\n", moniker)
			fmt.Printf("Chain ID: %s\n", cid)
			fmt.Printf("Home: %s\n", home)

			// Create directory structure
			dirs := []string{
				filepath.Join(home, "config"),
				filepath.Join(home, "data"),
				filepath.Join(home, "keys"),
			}
			for _, d := range dirs {
				if err := os.MkdirAll(d, 0755); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", d, err)
				}
			}

			// Generate validator consensus key (ed25519)
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return fmt.Errorf("failed to generate validator key: %w", err)
			}

			// Derive node ID from public key
			hash := sha256.Sum256(pub)
			nodeID := hex.EncodeToString(hash[:20])

			// Save validator key (priv_validator_key.json format)
			valKey := ValidatorKey{}
			valKey.Address = hex.EncodeToString(hash[:20])
			valKey.PubKey.Type = "tendermint/PubKeyEd25519"
			valKey.PubKey.Value = base64.StdEncoding.EncodeToString(pub)
			valKey.PrivKey.Type = "tendermint/PrivKeyEd25519"
			valKey.PrivKey.Value = base64.StdEncoding.EncodeToString(priv)

			valKeyPath := filepath.Join(home, "config", "priv_validator_key.json")
			valKeyData, _ := json.MarshalIndent(valKey, "", "  ")
			if err := os.WriteFile(valKeyPath, valKeyData, 0600); err != nil {
				return fmt.Errorf("failed to write validator key: %w", err)
			}

			// Save node key (node_key.json format)
			nodePub, nodePriv, _ := ed25519.GenerateKey(rand.Reader)
			nodeKey := map[string]interface{}{
				"priv_key": map[string]string{
					"type":  "tendermint/PrivKeyEd25519",
					"value": base64.StdEncoding.EncodeToString(nodePriv),
				},
				"pub_key": map[string]string{
					"type":  "tendermint/PubKeyEd25519",
					"value": base64.StdEncoding.EncodeToString(nodePub),
				},
			}
			nodeKeyPath := filepath.Join(home, "config", "node_key.json")
			nodeKeyData, _ := json.MarshalIndent(nodeKey, "", "  ")
			os.WriteFile(nodeKeyPath, nodeKeyData, 0600)

			// Generate genesis file
			genesis := generateGenesisJSON(cid, moniker, valKey)
			genesisPath := filepath.Join(home, "config", "genesis.json")
			genesisData, _ := json.MarshalIndent(genesis, "", "  ")
			if err := os.WriteFile(genesisPath, genesisData, 0644); err != nil {
				return fmt.Errorf("failed to write genesis: %w", err)
			}

			// Save node config
			nodeConfig := NodeConfig{
				Moniker:    moniker,
				ChainID:    cid,
				NodeID:     nodeID,
				ListenAddr: "tcp://0.0.0.0:26656",
				RPCAddr:    "tcp://0.0.0.0:26657",
				P2PAddr:    "tcp://0.0.0.0:26656",
			}
			configPath := filepath.Join(home, "config", "node_config.json")
			configData, _ := json.MarshalIndent(nodeConfig, "", "  ")
			os.WriteFile(configPath, configData, 0644)

			// Save validator state (empty initial state)
			valState := map[string]interface{}{
				"height":     "0",
				"round":      0,
				"step":       0,
				"signature":  "",
				"signbytes":  "",
			}
			valStatePath := filepath.Join(home, "data", "priv_validator_state.json")
			valStateData, _ := json.MarshalIndent(valState, "", "  ")
			os.WriteFile(valStatePath, valStateData, 0644)

			fmt.Println()
			fmt.Println("╔══════════════════════════════════════════════════════════════╗")
			fmt.Println("║         FTG ENERGY NODE INITIALIZED                         ║")
			fmt.Println("╚══════════════════════════════════════════════════════════════╝")
			fmt.Printf("  Node ID:     %s\n", nodeID)
			fmt.Printf("  Validator:   %s\n", valKey.Address)
			fmt.Printf("  Genesis:     %s\n", genesisPath)
			fmt.Printf("  Val Key:     %s\n", valKeyPath)
			fmt.Printf("  Node Key:    %s\n", nodeKeyPath)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  1. Generate treasury keys:  ftgd keys generate-all --home", home)
			fmt.Println("  2. Start the node:          ftgd start --home", home)

			return nil
		},
	}

	cmd.Flags().StringVar(&homeDir, "home", "/opt/ftg-data", "Home directory for node data")
	cmd.Flags().StringVar(&testnetChainID, "chain-id", "ftg-energy-testnet-1", "Chain ID")

	return cmd
}

// generateGenesisJSON creates the full genesis document
func generateGenesisJSON(chainID, moniker string, valKey ValidatorKey) map[string]interface{} {
	genesisTime := time.Now().UTC().Format(time.RFC3339Nano)

	return map[string]interface{}{
		"genesis_time": genesisTime,
		"chain_id":     chainID,
		"initial_height": "1",
		"consensus_params": map[string]interface{}{
			"block": map[string]interface{}{
				"max_bytes": "22020096",
				"max_gas":   "100000000",
			},
			"evidence": map[string]interface{}{
				"max_age_num_blocks": "100000",
				"max_age_duration":   "172800000000000",
				"max_bytes":          "1048576",
			},
			"validator": map[string]interface{}{
				"pub_key_types": []string{"ed25519"},
			},
		},
		"validators": []map[string]interface{}{
			{
				"address": valKey.Address,
				"pub_key": map[string]string{
					"type":  valKey.PubKey.Type,
					"value": valKey.PubKey.Value,
				},
				"power": "1000000",
				"name":  moniker,
			},
		},
		"app_state": map[string]interface{}{
			"energymint": map[string]interface{}{
				"params": map[string]interface{}{
					"tokens_per_mwh":      "100",
					"min_mwh_per_mint":    "1",
					"denom":              "uftg",
					"max_supply":         "0",
					"inflation_rate":     "0",
					"require_ipfs_proof": true,
				},
				"authorized_minters": []string{},
				"certificates":       []interface{}{},
				"total_minted":       "0",
			},
			"ftgburn": map[string]interface{}{
				"params": map[string]interface{}{
					"burn_address":        "ftg1000000000000000000000000000000000burn",
					"min_burn_amount":     "1000000",
					"merchant_protocol":   true,
					"protocol_fee_bps":    "0",
				},
				"total_burned":    "0",
				"burn_events":     []interface{}{},
			},
			"provenance": map[string]interface{}{
				"records": []interface{}{},
			},
			"treasury": map[string]interface{}{
				"minting_balance":   "0",
				"committed_balance": "0",
				"liquidity_balance": "0",
				"commitments":       []interface{}{},
			},
			"bank": map[string]interface{}{
				"supply": []map[string]string{
					{"denom": "uftg", "amount": "0"},
					{"denom": "ustake", "amount": "1000000000000"},
				},
				"balances": []interface{}{},
				"denom_metadata": []map[string]interface{}{
					{
						"description": "FTG Energy Token — 1 FTG = 10 kWh of verified stored energy",
						"denom_units": []map[string]interface{}{
							{"denom": "uftg", "exponent": 0, "aliases": []string{"microftg"}},
							{"denom": "mftg", "exponent": 3, "aliases": []string{"milliftg"}},
							{"denom": "ftg", "exponent": 6, "aliases": []string{"FTG"}},
						},
						"base":    "uftg",
						"display": "ftg",
						"name":    "FTG Energy",
						"symbol":  "FTG",
					},
				},
			},
			"staking": map[string]interface{}{
				"params": map[string]interface{}{
					"unbonding_time":     "1814400s",
					"max_validators":     100,
					"max_entries":        7,
					"historical_entries": 10000,
					"bond_denom":         "ustake",
				},
			},
			"mint": map[string]interface{}{
				"minter": map[string]interface{}{
					"inflation":         "0.000000000000000000",
					"annual_provisions": "0.000000000000000000",
				},
				"params": map[string]interface{}{
					"mint_denom":           "ustake",
					"inflation_rate_change": "0.000000000000000000",
					"inflation_max":        "0.000000000000000000",
					"inflation_min":        "0.000000000000000000",
					"goal_bonded":          "0.670000000000000000",
					"blocks_per_year":      "6311520",
				},
			},
		},
	}
}
