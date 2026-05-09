package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/energymint/types"
)

// Keeper of the energymint store
type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	logger   log.Logger
	bankKeeper types.BankKeeper
	authority  string
}

// NewKeeper creates a new energymint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
	bankKeeper types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		logger:     logger,
		bankKeeper: bankKeeper,
		authority:  authority,
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// MintFromCertificate validates a certificate and mints the corresponding FTG tokens
func (k Keeper) MintFromCertificate(
	ctx sdk.Context,
	sender sdk.AccAddress,
	cert types.Certificate,
	destinationWallet sdk.AccAddress,
) (*types.MintResponse, error) {
	k.Logger().Info("MintFromCertificate initiated",
		"sender", sender.String(),
		"external_id", cert.ExternalID,
		"mwh_quantity", cert.MwhQuantity.String(),
		"provenance_type", cert.ProvenanceType,
		"ipfs_hash", cert.IPFSHash,
	)

	// 1. Validate sender is an authorized minter
	if !k.IsAuthorizedMinter(ctx, sender) {
		return nil, fmt.Errorf("address %s is not an authorized minter", sender.String())
	}

	// 2. Validate provenance type
	params := k.GetParams(ctx)
	if !isAcceptedProvenance(params.AcceptedProvenanceTypes, cert.ProvenanceType) {
		return nil, fmt.Errorf("provenance type %s is not accepted", cert.ProvenanceType)
	}

	// 3. Validate IPFS hash if required
	if params.RequireIPFS && cert.IPFSHash == "" {
		return nil, fmt.Errorf("IPFS hash is required for minting")
	}

	// 4. Validate minimum MWh
	if cert.MwhQuantity.LT(params.MinCertificateMwh) {
		return nil, fmt.Errorf("certificate MWh (%s) below minimum (%s)",
			cert.MwhQuantity.String(), params.MinCertificateMwh.String())
	}

	// 5. Check certificate hasn't already been registered (prevent double-mint)
	if k.HasCertificate(ctx, cert.ExternalID) {
		return nil, fmt.Errorf("certificate %s has already been registered", cert.ExternalID)
	}

	// 6. Calculate tokens to mint
	// 1 MWh = 100 FTG tokens (each token = 10 kWh)
	// In micro-FTG: 1 MWh = 100 * 1,000,000 uftg = 100,000,000 uftg
	tokensToMint := cert.MwhQuantity.MulInt64(types.TokensPerMwh).MulInt64(types.MicroFTGPerToken).TruncateInt()

	// 7. Mint the tokens via bank module
	coins := sdk.NewCoins(sdk.NewCoin(types.TokenDenom, tokensToMint))
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return nil, fmt.Errorf("failed to mint coins: %w", err)
	}

	// 8. Send tokens to destination wallet
	if destinationWallet.Empty() {
		destinationWallet = sender
	}
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, destinationWallet, coins)
	if err != nil {
		return nil, fmt.Errorf("failed to send minted coins to destination: %w", err)
	}

	// 9. Record the certificate on-chain
	cert.CertificateID = k.GenerateCertificateID(ctx)
	cert.MintedTokens = tokensToMint
	cert.MintTimestamp = time.Now().UTC()
	cert.Owner = sender
	cert.Status = "active"
	k.SetCertificate(ctx, cert)

	// 10. Update totals
	k.IncrementTotalMinted(ctx, tokensToMint)
	k.IncrementTotalBackedMwh(ctx, cert.MwhQuantity)

	// 11. Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"ftg_mint",
			sdk.NewAttribute("certificate_id", cert.CertificateID),
			sdk.NewAttribute("external_id", cert.ExternalID),
			sdk.NewAttribute("mwh_quantity", cert.MwhQuantity.String()),
			sdk.NewAttribute("tokens_minted", tokensToMint.String()),
			sdk.NewAttribute("recipient", destinationWallet.String()),
			sdk.NewAttribute("provenance_type", cert.ProvenanceType),
			sdk.NewAttribute("ipfs_hash", cert.IPFSHash),
		),
	)

	k.Logger().Info("MintFromCertificate completed",
		"certificate_id", cert.CertificateID,
		"tokens_minted", tokensToMint.String(),
		"recipient", destinationWallet.String(),
	)

	return &types.MintResponse{
		CertificateID: cert.CertificateID,
		TokensMinted:  tokensToMint,
		Recipient:     destinationWallet,
	}, nil
}

// IsAuthorizedMinter checks if an address is authorized to mint
func (k Keeper) IsAuthorizedMinter(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.AuthorizedMinterKey(addr.String()))
}

// AddAuthorizedMinter adds an address to the authorized minters list
func (k Keeper) AddAuthorizedMinter(ctx sdk.Context, addr sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.AuthorizedMinterKey(addr.String()), []byte{1})
}

// RemoveAuthorizedMinter removes an address from the authorized minters list
func (k Keeper) RemoveAuthorizedMinter(ctx sdk.Context, addr sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.AuthorizedMinterKey(addr.String()))
}

// HasCertificate checks if a certificate with the given external ID already exists
func (k Keeper) HasCertificate(ctx sdk.Context, externalID string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.CertificateKey(externalID))
}

// SetCertificate stores a certificate in the KV store using JSON encoding
func (k Keeper) SetCertificate(ctx sdk.Context, cert types.Certificate) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&cert)
	store.Set(types.CertificateKey(cert.ExternalID), bz)
}

// GetCertificate retrieves a certificate by external ID
func (k Keeper) GetCertificate(ctx sdk.Context, externalID string) (types.Certificate, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CertificateKey(externalID))
	if bz == nil {
		return types.Certificate{}, false
	}
	var cert types.Certificate
	json.Unmarshal(bz, &cert)
	return cert, true
}

// GenerateCertificateID creates a unique certificate ID
func (k Keeper) GenerateCertificateID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.CertificateCountKey))
	var count uint64
	if bz != nil {
		count = sdk.BigEndianToUint64(bz)
	}
	count++
	store.Set([]byte(types.CertificateCountKey), sdk.Uint64ToBigEndian(count))
	return fmt.Sprintf("FTG-CERT-%06d", count)
}

// GetParams returns the current module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	json.Unmarshal(bz, &params)
	return params
}

// SetParams sets the module parameters
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&params)
	store.Set([]byte(types.ParamsKey), bz)
}

// IncrementTotalMinted adds to the total minted counter
func (k Keeper) IncrementTotalMinted(ctx sdk.Context, amount math.Int) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.TotalMintedKey))
	current := math.ZeroInt()
	if bz != nil {
		current.Unmarshal(bz)
	}
	newTotal := current.Add(amount)
	newBz, _ := newTotal.Marshal()
	store.Set([]byte(types.TotalMintedKey), newBz)
}

// GetTotalMinted returns the total tokens minted
func (k Keeper) GetTotalMinted(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.TotalMintedKey))
	if bz == nil {
		return math.ZeroInt()
	}
	var total math.Int
	total.Unmarshal(bz)
	return total
}

// IncrementTotalBackedMwh adds to the total backed MWh counter
func (k Keeper) IncrementTotalBackedMwh(ctx sdk.Context, amount math.LegacyDec) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.TotalBackedMwhKey))
	current := math.LegacyZeroDec()
	if bz != nil {
		current.Unmarshal(bz)
	}
	newTotal := current.Add(amount)
	bz, _ = newTotal.Marshal()
	store.Set([]byte(types.TotalBackedMwhKey), bz)
}

// GetTotalBackedMwh returns the total MWh backing all tokens
func (k Keeper) GetTotalBackedMwh(ctx sdk.Context) math.LegacyDec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.TotalBackedMwhKey))
	if bz == nil {
		return math.LegacyZeroDec()
	}
	var total math.LegacyDec
	total.Unmarshal(bz)
	return total
}

// helper
func isAcceptedProvenance(accepted []string, ptype string) bool {
	for _, a := range accepted {
		if a == ptype {
			return true
		}
	}
	return false
}
