package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/VelaFlux28/ftg-chain/x/burn/keeper"
	"github.com/VelaFlux28/ftg-chain/x/burn/types"
)

// mockBurnBankKeeper implements the BankKeeper interface for burn testing
type mockBurnBankKeeper struct {
	balances map[string]sdk.Coins
	burned   sdk.Coins
}

func newMockBurnBankKeeper() *mockBurnBankKeeper {
	return &mockBurnBankKeeper{
		balances: make(map[string]sdk.Coins),
		burned:   sdk.NewCoins(),
	}
}

func (m *mockBurnBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	m.burned = m.burned.Add(amounts...)
	return nil
}

func (m *mockBurnBankKeeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	key := senderAddr.String()
	m.balances[key] = m.balances[key].Sub(amt...)
	return nil
}

func (m *mockBurnBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := m.balances[addr.String()]
	return sdk.NewCoin(denom, coins.AmountOf(denom))
}

// fundAccount adds tokens to a test account
func (m *mockBurnBankKeeper) fundAccount(addr sdk.AccAddress, amount math.Int) {
	key := addr.String()
	m.balances[key] = sdk.NewCoins(sdk.NewCoin("uftg", amount))
}

// setupBurnKeeper creates a test burn keeper with an in-memory store
func setupBurnKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBurnBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bankKeeper := newMockBurnBankKeeper()
	authority := sdk.AccAddress([]byte("governance-authority")).String()

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		log.NewNopLogger(),
		bankKeeper,
		authority,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return k, ctx, bankKeeper
}

func TestBurnTokens_ManualBurn(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	sender := sdk.AccAddress([]byte("token-holder-addr"))
	burnAmount := math.NewInt(5_000_000) // 5 FTG

	// Fund the account
	bankKeeper.fundAccount(sender, math.NewInt(10_000_000)) // 10 FTG

	event, err := k.BurnTokens(ctx, sender, burnAmount, "manual", nil)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, "FTG-BURN-000001", event.BurnID)
	require.Equal(t, burnAmount, event.Amount)
	require.Equal(t, "manual", event.Source)

	// Verify bank keeper burned the tokens
	require.Equal(t, burnAmount, bankKeeper.burned.AmountOf("uftg"))

	// Verify stats updated
	stats := k.GetBurnStats(ctx)
	require.Equal(t, burnAmount, stats.TotalBurned)
	require.Equal(t, uint64(1), stats.TotalBurnEvents)
	require.Equal(t, burnAmount, stats.ManualBurned)
	require.True(t, stats.MerchantProtocolBurned.IsZero())
}

func TestBurnTokens_MerchantProtocol(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	merchant := sdk.AccAddress([]byte("merchant-contract"))
	burnAmount := math.NewInt(100_000_000) // 100 FTG

	// Authorize the merchant
	k.AddAuthorizedMerchant(ctx, merchant)

	// Fund the merchant account
	bankKeeper.fundAccount(merchant, math.NewInt(500_000_000)) // 500 FTG

	merchantData := &types.MerchantBurnData{
		EnergyRevenueUSD: math.LegacyNewDecWithPrec(500, 0), // $500
		DEXPurchasePrice: math.LegacyNewDecWithPrec(5, 0),    // $5.00
		VaultID:          "VAULT-001",
		GridOperator:     "ConEdison",
	}

	event, err := k.BurnTokens(ctx, merchant, burnAmount, "merchant_protocol", merchantData)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, "merchant_protocol", event.Source)
	require.NotNil(t, event.MerchantData)
	require.Equal(t, "VAULT-001", event.MerchantData.VaultID)

	// Verify stats
	stats := k.GetBurnStats(ctx)
	require.Equal(t, burnAmount, stats.MerchantProtocolBurned)
	require.True(t, stats.TotalEnergyRevenueUSD.Equal(math.LegacyNewDecWithPrec(500, 0)))
}

func TestBurnTokens_UnauthorizedMerchant(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	unauthorized := sdk.AccAddress([]byte("unauthorized-merc"))
	bankKeeper.fundAccount(unauthorized, math.NewInt(100_000_000))

	_, err := k.BurnTokens(ctx, unauthorized, math.NewInt(10_000_000), "merchant_protocol", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not authorized for merchant burns")
}

func TestBurnTokens_InsufficientBalance(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	sender := sdk.AccAddress([]byte("poor-holder-addr"))
	bankKeeper.fundAccount(sender, math.NewInt(500_000)) // 0.5 FTG

	// Try to burn 5 FTG (more than balance)
	_, err := k.BurnTokens(ctx, sender, math.NewInt(5_000_000), "manual", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient balance")
}

func TestBurnTokens_BelowMinimum(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	sender := sdk.AccAddress([]byte("token-holder-addr"))
	bankKeeper.fundAccount(sender, math.NewInt(10_000_000))

	// Default minimum is 1,000,000 uftg (1 FTG). Try to burn 0.5 FTG
	_, err := k.BurnTokens(ctx, sender, math.NewInt(500_000), "manual", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "below minimum")
}

func TestBurnTokens_InvalidSource(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	sender := sdk.AccAddress([]byte("token-holder-addr"))
	bankKeeper.fundAccount(sender, math.NewInt(10_000_000))

	_, err := k.BurnTokens(ctx, sender, math.NewInt(5_000_000), "fake_source", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid burn source")
}

func TestBurnTokens_RedemptionBurn(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	sender := sdk.AccAddress([]byte("redeemer-address"))
	burnAmount := math.NewInt(10_000_000) // 10 FTG = 100 kWh physical energy
	bankKeeper.fundAccount(sender, math.NewInt(50_000_000))

	event, err := k.BurnTokens(ctx, sender, burnAmount, "redemption", nil)
	require.NoError(t, err)
	require.Equal(t, "redemption", event.Source)

	stats := k.GetBurnStats(ctx)
	require.Equal(t, burnAmount, stats.RedemptionBurned)
}

func TestBurnStats_Accumulation(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	sender := sdk.AccAddress([]byte("multi-burner-addr"))
	bankKeeper.fundAccount(sender, math.NewInt(100_000_000)) // 100 FTG

	// Burn 3 times manually
	for i := 0; i < 3; i++ {
		_, err := k.BurnTokens(ctx, sender, math.NewInt(5_000_000), "manual", nil)
		require.NoError(t, err)
	}

	stats := k.GetBurnStats(ctx)
	require.Equal(t, math.NewInt(15_000_000), stats.TotalBurned)
	require.Equal(t, uint64(3), stats.TotalBurnEvents)
	require.Equal(t, math.NewInt(15_000_000), stats.ManualBurned)
}

func TestBurnIDGeneration(t *testing.T) {
	k, ctx, _ := setupBurnKeeper(t)

	id1 := k.GenerateBurnID(ctx)
	id2 := k.GenerateBurnID(ctx)
	id3 := k.GenerateBurnID(ctx)

	require.Equal(t, "FTG-BURN-000001", id1)
	require.Equal(t, "FTG-BURN-000002", id2)
	require.Equal(t, "FTG-BURN-000003", id3)
}

func TestMerchantProtocolBurn_FullFlow(t *testing.T) {
	k, ctx, bankKeeper := setupBurnKeeper(t)

	// Simulate the Automatic Merchant Protocol:
	// 1. Energy vault sells 100 kWh to grid for $50
	// 2. Revenue ($50) buys FTG on DEX at $5.00 = 10 FTG
	// 3. Those 10 FTG are burned permanently

	merchant := sdk.AccAddress([]byte("vault-merchant-01"))
	k.AddAuthorizedMerchant(ctx, merchant)
	tokensToBurn := math.NewInt(10_000_000) // 10 FTG in uftg
	bankKeeper.fundAccount(merchant, tokensToBurn)

	event, err := k.MerchantProtocolBurn(
		ctx,
		merchant,
		math.LegacyNewDecWithPrec(50, 0),  // $50 energy revenue
		math.LegacyNewDecWithPrec(5, 0),    // $5.00 per FTG
		tokensToBurn,
		"VAULT-EQUINOX-001",
		"ConEdison",
	)
	require.NoError(t, err)
	require.Equal(t, "merchant_protocol", event.Source)
	require.Equal(t, "VAULT-EQUINOX-001", event.MerchantData.VaultID)
	require.Equal(t, "ConEdison", event.MerchantData.GridOperator)

	// Verify the tokens were burned
	require.Equal(t, tokensToBurn, bankKeeper.burned.AmountOf("uftg"))
}
