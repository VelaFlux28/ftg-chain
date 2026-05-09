package app

// FTGApp is the main application struct for the FTG Energy blockchain.
// It wires together all Cosmos SDK standard modules plus our custom modules:
// - x/energymint: REC-backed token minting
// - x/ftgburn: Automatic Merchant Protocol (burn engine)
// - x/provenance: Certificate registry and audit trail
// - x/treasury: Segregated wallet management

const (
	AppName = "ftgd"
	Version = "0.1.0"
)

// Module account permissions
// These define which module accounts can mint, burn, or stake tokens
var maccPerms = map[string][]string{
	"energymint": {"minter"},          // Can mint new FTG tokens
	"ftgburn":    {"burner"},          // Can burn FTG tokens
	"treasury":   {},                  // Holds tokens but cannot mint/burn
	"fee_collector": {},               // Collects transaction fees
	"bonded_tokens_pool": {"burner", "staking"},
	"not_bonded_tokens_pool": {"burner", "staking"},
	"gov": {"burner"},
}

// OrderedModules defines the module initialization order
// Critical: energymint must initialize before treasury (treasury receives minted tokens)
var OrderedModules = []string{
	"auth",
	"bank",
	"staking",
	"slashing",
	"gov",
	"distribution",
	"evidence",
	"upgrade",
	"consensus",
	"energymint",  // Custom: REC-backed minting
	"ftgburn",     // Custom: Burn protocol
	"provenance",  // Custom: Certificate registry
	"treasury",    // Custom: Wallet segregation
	"genutil",
}

// BeginBlockOrder defines which modules run logic at the start of each block
var BeginBlockOrder = []string{
	"staking",
	"slashing",
	"evidence",
	"distribution",
	"energymint",
}

// EndBlockOrder defines which modules run logic at the end of each block
var EndBlockOrder = []string{
	"staking",
	"gov",
	"ftgburn",  // Process any pending merchant burns
}
