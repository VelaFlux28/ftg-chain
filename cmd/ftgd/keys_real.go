package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// KeyInfo represents a stored key
type KeyInfo struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	PubKeyHex  string `json:"pub_key_hex"`
	Type       string `json:"type"` // validator, treasury-minting, treasury-committed, treasury-liquidity
	CreatedAt  string `json:"created_at"`
}

// KeyStore manages keys on disk
type KeyStore struct {
	HomeDir string
}

func NewKeyStore(homeDir string) *KeyStore {
	keysDir := filepath.Join(homeDir, "keys")
	os.MkdirAll(keysDir, 0700)
	return &KeyStore{HomeDir: homeDir}
}

func (ks *KeyStore) keysDir() string {
	return filepath.Join(ks.HomeDir, "keys")
}

// GenerateKey creates a new ed25519 key pair and stores it
func (ks *KeyStore) GenerateKey(name, keyType string) (*KeyInfo, string, error) {
	// Generate ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Derive address from public key (sha256 hash, first 20 bytes, bech32-like hex for now)
	hash := sha256.Sum256(pub)
	address := "ftg1" + hex.EncodeToString(hash[:20])

	// Generate mnemonic-like entropy (24 words would require bip39 lib, using hex seed for now)
	seed := hex.EncodeToString(priv.Seed())

	keyInfo := &KeyInfo{
		Name:      name,
		Address:   address,
		PubKeyHex: hex.EncodeToString(pub),
		Type:      keyType,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Save key info (public)
	infoPath := filepath.Join(ks.keysDir(), name+".json")
	data, _ := json.MarshalIndent(keyInfo, "", "  ")
	if err := os.WriteFile(infoPath, data, 0600); err != nil {
		return nil, "", fmt.Errorf("failed to save key info: %w", err)
	}

	// Save private key (encrypted in production, plaintext for testnet)
	privPath := filepath.Join(ks.keysDir(), name+".key")
	if err := os.WriteFile(privPath, []byte(hex.EncodeToString(priv)), 0600); err != nil {
		return nil, "", fmt.Errorf("failed to save private key: %w", err)
	}

	return keyInfo, seed, nil
}

// ListKeys returns all stored keys
func (ks *KeyStore) ListKeys() ([]*KeyInfo, error) {
	var keys []*KeyInfo
	entries, err := os.ReadDir(ks.keysDir())
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(ks.keysDir(), entry.Name()))
			if err != nil {
				continue
			}
			var ki KeyInfo
			if json.Unmarshal(data, &ki) == nil {
				keys = append(keys, &ki)
			}
		}
	}
	return keys, nil
}

// GetKey returns a specific key by name
func (ks *KeyStore) GetKey(name string) (*KeyInfo, error) {
	data, err := os.ReadFile(filepath.Join(ks.keysDir(), name+".json"))
	if err != nil {
		return nil, fmt.Errorf("key '%s' not found", name)
	}
	var ki KeyInfo
	if err := json.Unmarshal(data, &ki); err != nil {
		return nil, err
	}
	return &ki, nil
}

// realKeysCmd replaces the placeholder keys command
func realKeysCmd() *cobra.Command {
	var homeDir string

	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Key management commands",
	}

	cmd.PersistentFlags().StringVar(&homeDir, "home", "/opt/ftg-data", "Home directory for key storage")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "generate-all",
			Short: "Generate all required keys (validator + treasury wallets)",
			RunE: func(cmd *cobra.Command, args []string) error {
				home, _ := cmd.Flags().GetString("home")
				ks := NewKeyStore(home)

				keyTypes := []struct {
					Name string
					Type string
				}{
					{"validator", "validator"},
					{"treasury-minting", "treasury-minting"},
					{"treasury-committed", "treasury-committed"},
					{"treasury-liquidity", "treasury-liquidity"},
				}

				fmt.Println("╔══════════════════════════════════════════════════════════════╗")
				fmt.Println("║         FTG ENERGY — KEY GENERATION                         ║")
				fmt.Println("║         ⚠️  SAVE THESE SEEDS SECURELY ⚠️                     ║")
				fmt.Println("╚══════════════════════════════════════════════════════════════╝")
				fmt.Println()

				for _, kt := range keyTypes {
					ki, seed, err := ks.GenerateKey(kt.Name, kt.Type)
					if err != nil {
						return fmt.Errorf("failed to generate %s key: %w", kt.Name, err)
					}
					fmt.Printf("━━━ %s ━━━\n", ki.Name)
					fmt.Printf("  Address:  %s\n", ki.Address)
					fmt.Printf("  Type:     %s\n", ki.Type)
					fmt.Printf("  Seed:     %s\n", seed)
					fmt.Println()
				}

				fmt.Println("╔══════════════════════════════════════════════════════════════╗")
				fmt.Println("║  All keys generated and stored in", home+"/keys/")
				fmt.Println("║  BACK UP THE SEEDS ABOVE — THEY CANNOT BE RECOVERED         ║")
				fmt.Println("╚══════════════════════════════════════════════════════════════╝")
				return nil
			},
		},
		&cobra.Command{
			Use:   "add [name]",
			Short: "Create a new key pair",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				home, _ := cmd.Flags().GetString("home")
				ks := NewKeyStore(home)
				ki, seed, err := ks.GenerateKey(args[0], "custom")
				if err != nil {
					return err
				}
				fmt.Printf("Key created: %s\n", ki.Name)
				fmt.Printf("Address: %s\n", ki.Address)
				fmt.Printf("Seed: %s\n", seed)
				fmt.Println("\n⚠️  Store your seed securely! It cannot be recovered.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all keys",
			RunE: func(cmd *cobra.Command, args []string) error {
				home, _ := cmd.Flags().GetString("home")
				ks := NewKeyStore(home)
				keys, err := ks.ListKeys()
				if err != nil {
					fmt.Println("No keys found. Run 'ftgd keys generate-all' first.")
					return nil
				}
				fmt.Println("═══ FTG Keys ═══")
				for _, k := range keys {
					fmt.Printf("  %-22s %s  [%s]\n", k.Name, k.Address, k.Type)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "show [name]",
			Short: "Show key details",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				home, _ := cmd.Flags().GetString("home")
				ks := NewKeyStore(home)
				ki, err := ks.GetKey(args[0])
				if err != nil {
					return err
				}
				fmt.Printf("Name:       %s\n", ki.Name)
				fmt.Printf("Address:    %s\n", ki.Address)
				fmt.Printf("Public Key: %s\n", ki.PubKeyHex)
				fmt.Printf("Type:       %s\n", ki.Type)
				fmt.Printf("Created:    %s\n", ki.CreatedAt)
				return nil
			},
		},
	)

	return cmd
}
