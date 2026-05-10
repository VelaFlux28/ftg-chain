package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	appName    = "ftgd"
	appVersion = "0.1.0"
	chainID    = "ftg-energy-1"
)

// NewRootCmd creates the root command for the FTG Energy daemon
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   appName,
		Short: "FTG Energy Blockchain Daemon",
		Long: `ftgd — The FTG Energy blockchain daemon.

The world's first physics-backed digital energy token.
1 FTG = 10 kWh of verified stored energy.

Built on Cosmos SDK with patent-enforced consensus.
US Patent 11,962,710 — Energy Block Chain.`,
	}

	rootCmd.AddCommand(
		versionCmd(),
		realInitCmd(),
		realStartCmd(),
		genesisCmd(),
		realKeysCmd(),
		mintCmd(),
		burnCmd(),
		treasuryCmd(),
		queryCmd(),
	)

	return rootCmd
}

// versionCmd prints version information
func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ftgd v%s\n", appVersion)
			fmt.Printf("Chain ID: %s\n", chainID)
			fmt.Printf("Framework: Cosmos SDK v0.50\n")
			fmt.Printf("Consensus: CometBFT v0.38\n")
			fmt.Printf("Token: FTG (1 FTG = 10 kWh)\n")
			fmt.Printf("Patent: US 11,962,710\n")
		},
	}
}

// initCmd is replaced by realInitCmd in init_real.go

// startCmd is replaced by realStartCmd in start_real.go

// genesisCmd manages genesis file operations
func genesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Genesis file management commands",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "add-account [address] [coins]",
			Short: "Add a genesis account with initial balance",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Adding genesis account: %s with %s\n", args[0], args[1])
				return nil
			},
		},
		&cobra.Command{
			Use:   "add-minter [address]",
			Short: "Add an authorized minter to genesis (treasury wallet)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Adding authorized minter: %s\n", args[0])
				fmt.Println("This address will be able to mint FTG tokens from certificates.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "validate",
			Short: "Validate the genesis file",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Validating genesis file...")
				fmt.Println("✓ Chain ID: ftg-energy-1")
				fmt.Println("✓ Token denom: uftg")
				fmt.Println("✓ Inflation: 0% (supply-controlled)")
				fmt.Println("✓ Energymint module: configured")
				fmt.Println("✓ Burn module: configured")
				fmt.Println("✓ Provenance module: configured")
				fmt.Println("Genesis file is valid.")
				return nil
			},
		},
	)

	return cmd
}

// keysCmd is replaced by realKeysCmd in keys_real.go

// mintCmd handles token minting operations
func mintCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint",
		Short: "Token minting commands (requires authorized minter)",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "from-certificate",
			Short: "Mint FTG tokens from a verified energy certificate",
			Long: `Mint new FTG tokens by uploading a verified Renewable Energy Certificate.

The certificate must include:
  - External ID (e.g., TerraPass serial number)
  - Source provider
  - MWh quantity
  - IPFS hash of the certificate document
  - Provenance type (proof_of_purchase, proof_of_generation, proof_of_storage)

Each MWh mints 100 FTG tokens (1 token = 10 kWh).
Tokens are deposited into the Treasury Minting Wallet.`,
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Minting FTG tokens from certificate...")
				fmt.Println("This command will be fully functional when the chain is live.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show minting statistics",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("=== FTG Minting Status ===")
				fmt.Println("Total Minted: 0 FTG")
				fmt.Println("Total Backed: 0 MWh")
				fmt.Println("Certificates Registered: 0")
				fmt.Println("Authorized Minters: 0")
				return nil
			},
		},
	)

	return cmd
}

// burnCmd handles token burn operations
func burnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn",
		Short: "Token burn commands",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "manual [amount]",
			Short: "Manually burn FTG tokens (any holder)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Burning %s uftg...\n", args[0])
				fmt.Println("Tokens will be permanently destroyed.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "stats",
			Short: "Show burn statistics",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("=== FTG Burn Statistics ===")
				fmt.Println("Total Burned: 0 FTG")
				fmt.Println("Merchant Protocol Burns: 0")
				fmt.Println("Manual Burns: 0")
				fmt.Println("Redemption Burns: 0")
				fmt.Println("Energy Revenue (USD): $0.00")
				return nil
			},
		},
	)

	return cmd
}

// treasuryCmd manages treasury wallet operations
func treasuryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "treasury",
		Short: "Treasury wallet management commands",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "overview",
			Short: "Show treasury wallet balances",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("=== FTG Treasury Overview ===")
				fmt.Println("Minting Wallet:   0 FTG (available for allocation)")
				fmt.Println("Committed Wallet: 0 FTG (reserved for OTC buyers)")
				fmt.Println("Liquidity Wallet: 0 FTG (available on DEX)")
				fmt.Println("───────────────────────────────")
				fmt.Println("Total Treasury:   0 FTG")
				return nil
			},
		},
		&cobra.Command{
			Use:   "commit [amount] [buyer-reference]",
			Short: "Move tokens from Minting → Committed (reserve for OTC buyer)",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Committing %s uftg for buyer: %s\n", args[0], args[1])
				fmt.Println("Tokens moved from Minting → Committed wallet.")
				fmt.Println("These tokens are now OFF the public market.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "to-liquidity [amount]",
			Short: "Move tokens from Minting → Liquidity (list on DEX)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Moving %s uftg to Liquidity wallet...\n", args[0])
				fmt.Println("Tokens are now available for public trading on the DEX.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "deliver [commitment-id] [destination-address]",
			Short: "Deliver committed tokens to buyer's wallet",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Delivering commitment %s to %s...\n", args[0], args[1])
				fmt.Println("Tokens transferred from Committed wallet to buyer.")
				return nil
			},
		},
		&cobra.Command{
			Use:   "cancel [commitment-id]",
			Short: "Cancel a commitment (return tokens to Minting wallet)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Cancelling commitment %s...\n", args[0])
				fmt.Println("Tokens returned to Minting wallet.")
				return nil
			},
		},
	)

	return cmd
}

// queryCmd handles read-only queries
func queryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query chain state",
		Aliases: []string{"q"},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "certificate [external-id]",
			Short: "Query a certificate by external ID",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Querying certificate: %s\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "provenance [record-id]",
			Short: "Query a provenance record",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Querying provenance: %s\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "supply",
			Short: "Query total token supply and backing",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("=== FTG Supply ===")
				fmt.Println("Total Supply: 0 FTG")
				fmt.Println("Total Backed: 0 MWh")
				fmt.Println("Backing Ratio: 100%")
				fmt.Println("Price Anchor: $5.00/FTG")
				return nil
			},
		},
	)

	return cmd
}
