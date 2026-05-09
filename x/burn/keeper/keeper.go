package keeper

import (
	"fmt"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/burn/types"
)

// BankKeeper defines the expected bank module interface for the burn module
type BankKeeper interface {
	BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// Keeper of the burn store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	logger     log.Logger
	bankKeeper BankKeeper
	authority  string
}

// NewKeeper creates a new burn Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
	bankKeeper BankKeeper,
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

// BurnTokens executes a token burn
// This is the core deflationary mechanism — tokens are permanently destroyed
func (k Keeper) BurnTokens(
	ctx sdk.Context,
	sender sdk.AccAddress,
	amount math.Int,
	source string,
	merchantData *types.MerchantBurnData,
) (*types.BurnEvent, error) {
	k.Logger().Info("BurnTokens initiated",
		"sender", sender.String(),
		"amount", amount.String(),
		"source", source,
	)

	// 1. Validate source type
	if source != "merchant_protocol" && source != "manual" && source != "redemption" {
		return nil, fmt.Errorf("invalid burn source: %s", source)
	}

	// 2. For merchant protocol burns, validate sender is authorized
	if source == "merchant_protocol" {
		if !k.IsAuthorizedMerchant(ctx, sender) {
			return nil, fmt.Errorf("address %s is not authorized for merchant burns", sender.String())
		}
		params := k.GetParams(ctx)
		if !params.MerchantProtocolEnabled {
			return nil, fmt.Errorf("merchant protocol is currently disabled")
		}
	}

	// 3. Validate minimum burn amount
	params := k.GetParams(ctx)
	if amount.LT(params.MinBurnAmount) {
		return nil, fmt.Errorf("burn amount (%s) below minimum (%s)",
			amount.String(), params.MinBurnAmount.String())
	}

	// 4. Check sender has sufficient balance
	balance := k.bankKeeper.GetBalance(ctx, sender, "uftg")
	if balance.Amount.LT(amount) {
		return nil, fmt.Errorf("insufficient balance: has %s, needs %s",
			balance.Amount.String(), amount.String())
	}

	// 5. Send tokens from sender to module account
	coins := sdk.NewCoins(sdk.NewCoin("uftg", amount))
	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, coins)
	if err != nil {
		return nil, fmt.Errorf("failed to send coins to burn module: %w", err)
	}

	// 6. Burn the tokens (permanently destroy)
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return nil, fmt.Errorf("failed to burn coins: %w", err)
	}

	// 7. Record the burn event
	burnEvent := types.BurnEvent{
		BurnID:       k.GenerateBurnID(ctx),
		Amount:       amount,
		Source:       source,
		Sender:       sender,
		Timestamp:    time.Now().UTC(),
		MerchantData: merchantData,
	}
	k.SetBurnEvent(ctx, burnEvent)

	// 8. Update statistics
	k.UpdateBurnStats(ctx, amount, source, merchantData)

	// 9. Emit event
	attrs := []sdk.Attribute{
		sdk.NewAttribute("burn_id", burnEvent.BurnID),
		sdk.NewAttribute("amount", amount.String()),
		sdk.NewAttribute("source", source),
		sdk.NewAttribute("sender", sender.String()),
	}
	if merchantData != nil {
		attrs = append(attrs,
			sdk.NewAttribute("energy_revenue_usd", merchantData.EnergyRevenueUSD.String()),
			sdk.NewAttribute("dex_purchase_price", merchantData.DEXPurchasePrice.String()),
		)
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("ftg_burn", attrs...))

	k.Logger().Info("BurnTokens completed",
		"burn_id", burnEvent.BurnID,
		"amount_burned", amount.String(),
		"source", source,
	)

	return &burnEvent, nil
}

// MerchantProtocolBurn executes the Automatic Merchant Protocol
// Energy revenue → Buy FTG on DEX → Burn to null address
// This is the automated deflationary engine
func (k Keeper) MerchantProtocolBurn(
	ctx sdk.Context,
	liquidityContract sdk.AccAddress,
	energyRevenueUSD sdk.Dec,
	dexPurchasePrice sdk.Dec,
	tokensPurchased math.Int,
	vaultID string,
	gridOperator string,
) (*types.BurnEvent, error) {
	merchantData := &types.MerchantBurnData{
		EnergyRevenueUSD: energyRevenueUSD,
		DEXPurchasePrice: dexPurchasePrice,
		VaultID:          vaultID,
		GridOperator:     gridOperator,
	}

	return k.BurnTokens(ctx, liquidityContract, tokensPurchased, "merchant_protocol", merchantData)
}

// IsAuthorizedMerchant checks if an address is authorized for merchant burns
func (k Keeper) IsAuthorizedMerchant(ctx sdk.Context, addr sdk.AccAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.AuthorizedMerchantKey(addr.String()))
}

// AddAuthorizedMerchant adds an address to the authorized merchant list
func (k Keeper) AddAuthorizedMerchant(ctx sdk.Context, addr sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.AuthorizedMerchantKey(addr.String()), []byte{1})
}

// SetBurnEvent stores a burn event
func (k Keeper) SetBurnEvent(ctx sdk.Context, event types.BurnEvent) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&event)
	store.Set(types.BurnEventKey(event.BurnID), bz)
}

// GenerateBurnID creates a unique burn event ID
func (k Keeper) GenerateBurnID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.BurnEventCountKey))
	var count uint64
	if bz != nil {
		count = sdk.BigEndianToUint64(bz)
	}
	count++
	store.Set([]byte(types.BurnEventCountKey), sdk.Uint64ToBigEndian(count))
	return fmt.Sprintf("FTG-BURN-%06d", count)
}

// GetParams returns the current module parameters
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	// For now return defaults — will be stored in KV store
	return types.DefaultParams()
}

// UpdateBurnStats updates cumulative burn statistics
func (k Keeper) UpdateBurnStats(ctx sdk.Context, amount math.Int, source string, merchantData *types.MerchantBurnData) {
	stats := k.GetBurnStats(ctx)
	stats.TotalBurned = stats.TotalBurned.Add(amount)
	stats.TotalBurnEvents++

	switch source {
	case "merchant_protocol":
		stats.MerchantProtocolBurned = stats.MerchantProtocolBurned.Add(amount)
		if merchantData != nil {
			stats.TotalEnergyRevenueUSD = stats.TotalEnergyRevenueUSD.Add(merchantData.EnergyRevenueUSD)
		}
	case "manual":
		stats.ManualBurned = stats.ManualBurned.Add(amount)
	case "redemption":
		stats.RedemptionBurned = stats.RedemptionBurned.Add(amount)
	}

	k.SetBurnStats(ctx, stats)
}

// GetBurnStats returns cumulative burn statistics
func (k Keeper) GetBurnStats(ctx sdk.Context) types.BurnStats {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.BurnStatsKey))
	if bz == nil {
		return types.BurnStats{
			TotalBurned:            math.ZeroInt(),
			MerchantProtocolBurned: math.ZeroInt(),
			ManualBurned:           math.ZeroInt(),
			RedemptionBurned:       math.ZeroInt(),
			TotalEnergyRevenueUSD:  sdk.ZeroDec(),
		}
	}
	var stats types.BurnStats
	k.cdc.MustUnmarshal(bz, &stats)
	return stats
}

// SetBurnStats stores burn statistics
func (k Keeper) SetBurnStats(ctx sdk.Context, stats types.BurnStats) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&stats)
	store.Set([]byte(types.BurnStatsKey), bz)
}
