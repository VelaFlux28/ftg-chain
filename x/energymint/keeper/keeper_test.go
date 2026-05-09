package keeper_test

import (
	"fmt"
	"testing"
	"time"

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

	"github.com/VelaFlux28/ftg-chain/x/energymint/keeper"
	"github.com/VelaFlux28/ftg-chain/x/energymint/types"
)

// mockBankKeeper implements the BankKeeper interface for testing
type mockBankKeeper struct {
	balances map[string]sdk.Coins
	minted   sdk.Coins
	burned   sdk.Coins
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdk.Coins),
		minted:   sdk.NewCoins(),
		burned:   sdk.NewCoins(),
	}
}

func (m *mockBankKeeper) MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	m.minted = m.minted.Add(amounts...)
	key := "module/" + moduleName
	m.balances[key] = m.balances[key].Add(amounts...)
	return nil
}

func (m *mockBankKeeper) BurnCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error {
	m.burned = m.burned.Add(amounts...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	key := "module/" + senderModule
	m.balances[key] = m.balances[key].Sub(amt...)
	addrKey := recipientAddr.String()
	m.balances[addrKey] = m.balances[addrKey].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	addrKey := senderAddr.String()
	m.balances[addrKey] = m.balances[addrKey].Sub(amt...)
	key := "module/" + recipientModule
	m.balances[key] = m.balances[key].Add(amt...)
	return nil
}

func (m *mockBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := m.balances[addr.String()]
	return sdk.NewCoin(denom, coins.AmountOf(denom))
}

func (m *mockBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return m.balances[addr.String()]
}

// setupKeeper creates a test keeper with an in-memory store
func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bankKeeper := newMockBankKeeper()
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

func TestMintFromCertificate_Success(t *testing.T) {
	k, ctx, bankKeeper := setupKeeper(t)

	// Create and authorize a minter
	minter := sdk.AccAddress([]byte("minter-address-1234"))
	k.AddAuthorizedMinter(ctx, minter)

	// Create a certificate for 10 MWh
	cert := types.Certificate{
		ExternalID:     "REC-2026-001",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(10, 0), // 10 MWh
		GenerationDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PurchaseDate:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		IPFSHash:       "QmTestHash123456789",
		ProvenanceType: "proof_of_purchase",
	}

	resp, err := k.MintFromCertificate(ctx, minter, cert, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 10 MWh × 100 tokens/MWh × 1,000,000 uftg/token = 1,000,000,000 uftg
	expectedTokens := math.NewInt(1_000_000_000)
	require.Equal(t, expectedTokens, resp.TokensMinted)
	require.Equal(t, "FTG-CERT-000001", resp.CertificateID)

	// Verify bank keeper received the mint call
	require.Equal(t, expectedTokens, bankKeeper.minted.AmountOf("uftg"))

	// Verify totals were updated
	totalMinted := k.GetTotalMinted(ctx)
	require.Equal(t, expectedTokens, totalMinted)

	totalBacked := k.GetTotalBackedMwh(ctx)
	require.True(t, totalBacked.Equal(math.LegacyNewDecWithPrec(10, 0)))
}

func TestMintFromCertificate_617MWh_GenesisScenario(t *testing.T) {
	k, ctx, bankKeeper := setupKeeper(t)

	minter := sdk.AccAddress([]byte("nick-treasury-addr"))
	k.AddAuthorizedMinter(ctx, minter)

	// Genesis scenario: 617 MWh of existing RECs
	cert := types.Certificate{
		ExternalID:     "GENESIS-REC-617MWH",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(617, 0), // 617 MWh
		GenerationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		PurchaseDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
		IPFSHash:       "QmGenesisRECDocumentHash",
		ProvenanceType: "proof_of_purchase",
	}

	resp, err := k.MintFromCertificate(ctx, minter, cert, nil)
	require.NoError(t, err)

	// 617 MWh × 100 tokens/MWh = 61,700 FTG
	// 61,700 × 1,000,000 = 61,700,000,000 uftg
	expectedTokens := math.NewInt(61_700_000_000)
	require.Equal(t, expectedTokens, resp.TokensMinted)

	// Verify the correct amount was minted
	require.Equal(t, expectedTokens, bankKeeper.minted.AmountOf("uftg"))
}

func TestMintFromCertificate_UnauthorizedMinter(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Don't authorize the minter
	unauthorized := sdk.AccAddress([]byte("unauthorized-addr"))

	cert := types.Certificate{
		ExternalID:     "REC-2026-002",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(5, 0),
		GenerationDate: time.Now(),
		PurchaseDate:   time.Now(),
		IPFSHash:       "QmTestHash",
		ProvenanceType: "proof_of_purchase",
	}

	_, err := k.MintFromCertificate(ctx, unauthorized, cert, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not an authorized minter")
}

func TestMintFromCertificate_DoubleMintPrevention(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))
	k.AddAuthorizedMinter(ctx, minter)

	cert := types.Certificate{
		ExternalID:     "REC-DOUBLE-001",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(5, 0),
		GenerationDate: time.Now(),
		PurchaseDate:   time.Now(),
		IPFSHash:       "QmTestHash",
		ProvenanceType: "proof_of_purchase",
	}

	// First mint should succeed
	_, err := k.MintFromCertificate(ctx, minter, cert, nil)
	require.NoError(t, err)

	// Second mint with same external ID should fail
	_, err = k.MintFromCertificate(ctx, minter, cert, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already been registered")
}

func TestMintFromCertificate_BelowMinimumMwh(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))
	k.AddAuthorizedMinter(ctx, minter)

	cert := types.Certificate{
		ExternalID:     "REC-SMALL-001",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(5, 1), // 0.5 MWh — below minimum of 1 MWh
		GenerationDate: time.Now(),
		PurchaseDate:   time.Now(),
		IPFSHash:       "QmTestHash",
		ProvenanceType: "proof_of_purchase",
	}

	_, err := k.MintFromCertificate(ctx, minter, cert, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "below minimum")
}

func TestMintFromCertificate_MissingIPFS(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))
	k.AddAuthorizedMinter(ctx, minter)

	cert := types.Certificate{
		ExternalID:     "REC-NOIPFS-001",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(5, 0),
		GenerationDate: time.Now(),
		PurchaseDate:   time.Now(),
		IPFSHash:       "", // Missing IPFS hash
		ProvenanceType: "proof_of_purchase",
	}

	_, err := k.MintFromCertificate(ctx, minter, cert, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "IPFS hash is required")
}

func TestMintFromCertificate_InvalidProvenanceType(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))
	k.AddAuthorizedMinter(ctx, minter)

	cert := types.Certificate{
		ExternalID:     "REC-BADTYPE-001",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(5, 0),
		GenerationDate: time.Now(),
		PurchaseDate:   time.Now(),
		IPFSHash:       "QmTestHash",
		ProvenanceType: "fake_provenance", // Invalid type
	}

	_, err := k.MintFromCertificate(ctx, minter, cert, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not accepted")
}

func TestMintFromCertificate_CustomDestination(t *testing.T) {
	k, ctx, bankKeeper := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))
	destination := sdk.AccAddress([]byte("buyer-wallet-addr"))
	k.AddAuthorizedMinter(ctx, minter)

	cert := types.Certificate{
		ExternalID:     "REC-DEST-001",
		Source:         "TerraPass",
		MwhQuantity:    math.LegacyNewDecWithPrec(1, 0), // 1 MWh
		GenerationDate: time.Now(),
		PurchaseDate:   time.Now(),
		IPFSHash:       "QmTestHash",
		ProvenanceType: "proof_of_purchase",
	}

	resp, err := k.MintFromCertificate(ctx, minter, cert, destination)
	require.NoError(t, err)
	require.Equal(t, destination, resp.Recipient)

	// Verify tokens went to the destination wallet
	destBalance := bankKeeper.GetBalance(ctx, destination, "uftg")
	require.Equal(t, math.NewInt(100_000_000), destBalance.Amount) // 1 MWh = 100 FTG = 100,000,000 uftg
}

func TestAuthorizedMinter_AddRemove(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))

	// Initially not authorized
	require.False(t, k.IsAuthorizedMinter(ctx, minter))

	// Add minter
	k.AddAuthorizedMinter(ctx, minter)
	require.True(t, k.IsAuthorizedMinter(ctx, minter))

	// Remove minter
	k.RemoveAuthorizedMinter(ctx, minter)
	require.False(t, k.IsAuthorizedMinter(ctx, minter))
}

func TestParams_DefaultAndCustom(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Default params
	params := k.GetParams(ctx)
	require.True(t, params.RequireIPFS)
	require.Equal(t, math.LegacyNewDecWithPrec(1, 0), params.MinCertificateMwh)
	require.Len(t, params.AcceptedProvenanceTypes, 3)

	// Set custom params
	customParams := types.Params{
		AuthorizedMinters:      []sdk.AccAddress{},
		MinCertificateMwh:      math.LegacyNewDecWithPrec(5, 0), // 5 MWh minimum
		RequireIPFS:            false,
		AcceptedProvenanceTypes: []string{"proof_of_purchase"},
	}
	k.SetParams(ctx, customParams)

	// Verify custom params
	retrieved := k.GetParams(ctx)
	require.False(t, retrieved.RequireIPFS)
	require.Equal(t, math.LegacyNewDecWithPrec(5, 0), retrieved.MinCertificateMwh)
	require.Len(t, retrieved.AcceptedProvenanceTypes, 1)
}

func TestCertificateIDGeneration(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	id1 := k.GenerateCertificateID(ctx)
	id2 := k.GenerateCertificateID(ctx)
	id3 := k.GenerateCertificateID(ctx)

	require.Equal(t, "FTG-CERT-000001", id1)
	require.Equal(t, "FTG-CERT-000002", id2)
	require.Equal(t, "FTG-CERT-000003", id3)
}

func TestTotalMintedAccumulation(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	minter := sdk.AccAddress([]byte("minter-address-1234"))
	k.AddAuthorizedMinter(ctx, minter)

	// Mint 3 certificates
	for i, mwh := range []int64{10, 20, 30} {
		cert := types.Certificate{
			ExternalID:     fmt.Sprintf("REC-ACCUM-%03d", i+1),
			Source:         "TerraPass",
			MwhQuantity:    math.LegacyNewDecWithPrec(mwh, 0),
			GenerationDate: time.Now(),
			PurchaseDate:   time.Now(),
			IPFSHash:       "QmTestHash",
			ProvenanceType: "proof_of_purchase",
		}
		_, err := k.MintFromCertificate(ctx, minter, cert, nil)
		require.NoError(t, err)
	}

	// Total should be 60 MWh = 6,000 FTG = 6,000,000,000 uftg
	totalMinted := k.GetTotalMinted(ctx)
	require.Equal(t, math.NewInt(6_000_000_000), totalMinted)

	totalBacked := k.GetTotalBackedMwh(ctx)
	require.True(t, totalBacked.Equal(math.LegacyNewDecWithPrec(60, 0)))
}
