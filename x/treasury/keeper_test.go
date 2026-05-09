package treasury_test

import (
	"fmt"
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

	"github.com/VelaFlux28/ftg-chain/x/treasury"
)

// mockTreasuryBankKeeper implements the BankKeeper interface for treasury testing
type mockTreasuryBankKeeper struct {
	balances map[string]sdk.Coins
}

func newMockTreasuryBankKeeper() *mockTreasuryBankKeeper {
	return &mockTreasuryBankKeeper{
		balances: make(map[string]sdk.Coins),
	}
}

func (m *mockTreasuryBankKeeper) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	fromKey := fromAddr.String()
	toKey := toAddr.String()

	// Check sufficient balance
	fromBalance := m.balances[fromKey]
	for _, coin := range amt {
		if fromBalance.AmountOf(coin.Denom).LT(coin.Amount) {
			return fmt.Errorf("insufficient funds")
		}
	}

	m.balances[fromKey] = m.balances[fromKey].Sub(amt...)
	m.balances[toKey] = m.balances[toKey].Add(amt...)
	return nil
}

func (m *mockTreasuryBankKeeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	coins := m.balances[addr.String()]
	return sdk.NewCoin(denom, coins.AmountOf(denom))
}

func (m *mockTreasuryBankKeeper) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return m.balances[addr.String()]
}

func (m *mockTreasuryBankKeeper) fundAccount(addr sdk.AccAddress, amount math.Int) {
	key := addr.String()
	m.balances[key] = sdk.NewCoins(sdk.NewCoin("uftg", amount))
}

// Test addresses
var (
	ownerAddr     = sdk.AccAddress([]byte("nick-owner-address"))
	mintingAddr   = sdk.AccAddress([]byte("minting-wallet-adr"))
	committedAddr = sdk.AccAddress([]byte("committed-wallet-a"))
	liquidityAddr = sdk.AccAddress([]byte("liquidity-wallet-a"))
	buyerAddr     = sdk.AccAddress([]byte("buyer-wallet-addr"))
	nonOwnerAddr  = sdk.AccAddress([]byte("non-owner-address"))
)

func setupTreasuryKeeper(t *testing.T) (treasury.Keeper, sdk.Context, *mockTreasuryBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(treasury.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bankKeeper := newMockTreasuryBankKeeper()

	config := treasury.TreasuryConfig{
		MintingWalletAddress:   mintingAddr,
		CommittedWalletAddress: committedAddr,
		LiquidityWalletAddress: liquidityAddr,
		OwnerAddress:           ownerAddr,
	}

	k := treasury.NewKeeper(cdc, storeKey, log.NewNopLogger(), bankKeeper, config)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	return k, ctx, bankKeeper
}

func TestTransferMintingToCommitted_Success(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	// Fund the minting wallet with 61,700 FTG (Genesis supply)
	genesisSupply := math.NewInt(61_700_000_000) // 61,700 FTG in uftg
	bankKeeper.fundAccount(mintingAddr, genesisSupply)

	// Commit 2,000,000 FTG for a $10M OTC buyer
	commitAmount := math.NewInt(2_000_000_000_000) // 2M FTG in uftg
	bankKeeper.fundAccount(mintingAddr, commitAmount) // Fund with enough

	req := treasury.TransferToCommitted{
		Amount:         math.NewInt(10_000_000_000), // 10,000 FTG
		BuyerReference: "OTC-BUYER-ALPHA",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0), // $5.00
	}

	commitment, err := k.TransferMintingToCommitted(ctx, ownerAddr, req)
	require.NoError(t, err)
	require.NotNil(t, commitment)
	require.Equal(t, "FTG-COMMIT-000001", commitment.CommitmentID)
	require.Equal(t, "OTC-BUYER-ALPHA", commitment.BuyerReference)
	require.Equal(t, "funded", commitment.Status)

	// Verify balances moved
	mintingBalance := bankKeeper.GetBalance(ctx, mintingAddr, "uftg")
	committedBalance := bankKeeper.GetBalance(ctx, committedAddr, "uftg")
	require.Equal(t, req.Amount, committedBalance.Amount)
	require.True(t, mintingBalance.Amount.LT(commitAmount)) // Reduced
}

func TestTransferMintingToCommitted_Unauthorized(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(100_000_000))

	req := treasury.TransferToCommitted{
		Amount:         math.NewInt(10_000_000),
		BuyerReference: "UNAUTHORIZED-BUYER",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0),
	}

	_, err := k.TransferMintingToCommitted(ctx, nonOwnerAddr, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

func TestTransferMintingToCommitted_InsufficientBalance(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	// Fund with only 100 FTG
	bankKeeper.fundAccount(mintingAddr, math.NewInt(100_000_000))

	req := treasury.TransferToCommitted{
		Amount:         math.NewInt(1_000_000_000), // 1,000 FTG — more than available
		BuyerReference: "BIG-BUYER",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0),
	}

	_, err := k.TransferMintingToCommitted(ctx, ownerAddr, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient minting balance")
}

func TestTransferMintingToLiquidity_Success(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(50_000_000_000)) // 50,000 FTG

	req := treasury.TransferToLiquidity{
		Amount: math.NewInt(10_000_000_000), // 10,000 FTG
	}

	err := k.TransferMintingToLiquidity(ctx, ownerAddr, req)
	require.NoError(t, err)

	// Verify liquidity wallet received tokens
	liquidityBalance := bankKeeper.GetBalance(ctx, liquidityAddr, "uftg")
	require.Equal(t, req.Amount, liquidityBalance.Amount)
}

func TestTransferMintingToLiquidity_Unauthorized(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(50_000_000_000))

	req := treasury.TransferToLiquidity{
		Amount: math.NewInt(10_000_000_000),
	}

	err := k.TransferMintingToLiquidity(ctx, nonOwnerAddr, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}

func TestDeliverCommitment_FullFlow(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	// Fund minting wallet
	bankKeeper.fundAccount(mintingAddr, math.NewInt(100_000_000_000))

	// Step 1: Create commitment
	commitReq := treasury.TransferToCommitted{
		Amount:         math.NewInt(10_000_000_000), // 10,000 FTG
		BuyerReference: "OTC-BUYER-BETA",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0),
	}

	commitment, err := k.TransferMintingToCommitted(ctx, ownerAddr, commitReq)
	require.NoError(t, err)
	require.Equal(t, "funded", commitment.Status)

	// Step 2: Deliver to buyer
	deliverReq := treasury.DeliverCommitment{
		CommitmentID:      commitment.CommitmentID,
		DestinationWallet: buyerAddr,
	}

	err = k.DeliverCommitment(ctx, ownerAddr, deliverReq)
	require.NoError(t, err)

	// Verify buyer received tokens
	buyerBalance := bankKeeper.GetBalance(ctx, buyerAddr, "uftg")
	require.Equal(t, commitReq.Amount, buyerBalance.Amount)

	// Verify commitment status updated
	updated, found := k.GetCommitment(ctx, commitment.CommitmentID)
	require.True(t, found)
	require.Equal(t, "delivered", updated.Status)
	require.True(t, updated.DeliveredAt > 0)
}

func TestDeliverCommitment_AlreadyDelivered(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(100_000_000_000))

	// Create and deliver
	commitReq := treasury.TransferToCommitted{
		Amount:         math.NewInt(10_000_000_000),
		BuyerReference: "BUYER-ONCE",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0),
	}
	commitment, _ := k.TransferMintingToCommitted(ctx, ownerAddr, commitReq)

	deliverReq := treasury.DeliverCommitment{
		CommitmentID:      commitment.CommitmentID,
		DestinationWallet: buyerAddr,
	}
	err := k.DeliverCommitment(ctx, ownerAddr, deliverReq)
	require.NoError(t, err)

	// Try to deliver again — should fail
	err = k.DeliverCommitment(ctx, ownerAddr, deliverReq)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not in funded status")
}

func TestCancelCommitment_Success(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(100_000_000_000))

	// Create commitment
	commitReq := treasury.TransferToCommitted{
		Amount:         math.NewInt(10_000_000_000),
		BuyerReference: "BUYER-CANCEL",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0),
	}
	commitment, _ := k.TransferMintingToCommitted(ctx, ownerAddr, commitReq)

	// Cancel it
	cancelReq := treasury.CancelCommitment{
		CommitmentID: commitment.CommitmentID,
		Reason:       "Buyer withdrew from deal",
	}
	err := k.CancelCommitmentOp(ctx, ownerAddr, cancelReq)
	require.NoError(t, err)

	// Verify tokens returned to minting wallet
	mintingBalance := bankKeeper.GetBalance(ctx, mintingAddr, "uftg")
	require.Equal(t, math.NewInt(100_000_000_000), mintingBalance.Amount) // Full amount restored

	// Verify commitment status
	updated, found := k.GetCommitment(ctx, commitment.CommitmentID)
	require.True(t, found)
	require.Equal(t, "cancelled", updated.Status)
}

func TestCancelCommitment_AlreadyDelivered(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(100_000_000_000))

	// Create and deliver
	commitReq := treasury.TransferToCommitted{
		Amount:         math.NewInt(10_000_000_000),
		BuyerReference: "BUYER-NODELETE",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0),
	}
	commitment, _ := k.TransferMintingToCommitted(ctx, ownerAddr, commitReq)

	deliverReq := treasury.DeliverCommitment{
		CommitmentID:      commitment.CommitmentID,
		DestinationWallet: buyerAddr,
	}
	_ = k.DeliverCommitment(ctx, ownerAddr, deliverReq)

	// Try to cancel delivered commitment — should fail
	cancelReq := treasury.CancelCommitment{
		CommitmentID: commitment.CommitmentID,
		Reason:       "Changed mind",
	}
	err := k.CancelCommitmentOp(ctx, ownerAddr, cancelReq)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be cancelled")
}

func TestTreasuryOverview(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	bankKeeper.fundAccount(mintingAddr, math.NewInt(50_000_000_000))
	bankKeeper.fundAccount(committedAddr, math.NewInt(10_000_000_000))
	bankKeeper.fundAccount(liquidityAddr, math.NewInt(5_000_000_000))

	overview := k.GetTreasuryOverview(ctx)
	require.Equal(t, "50000000000", overview["minting_balance"])
	require.Equal(t, "10000000000", overview["committed_balance"])
	require.Equal(t, "5000000000", overview["liquidity_balance"])
}

func TestCommitmentIDGeneration(t *testing.T) {
	k, ctx, _ := setupTreasuryKeeper(t)

	id1 := k.GenerateCommitmentID(ctx)
	id2 := k.GenerateCommitmentID(ctx)
	id3 := k.GenerateCommitmentID(ctx)

	require.Equal(t, "FTG-COMMIT-000001", id1)
	require.Equal(t, "FTG-COMMIT-000002", id2)
	require.Equal(t, "FTG-COMMIT-000003", id3)
}

func TestFullOTCFlow_10MillionDollarDeal(t *testing.T) {
	k, ctx, bankKeeper := setupTreasuryKeeper(t)

	// Simulate the full $10M OTC flow from our architecture discussions:
	// 1. Nick buys 20,000 MWh of RECs ($160K)
	// 2. Mints 2,000,000 FTG tokens into Treasury
	// 3. Commits tokens for OTC buyer
	// 4. Delivers to buyer's wallet

	// Step 1: After minting, 2M FTG in minting wallet
	twoMillionFTG := math.NewInt(2_000_000_000_000) // 2M FTG in uftg
	bankKeeper.fundAccount(mintingAddr, twoMillionFTG)

	// Step 2: Commit for OTC buyer at $5.00/token
	commitReq := treasury.TransferToCommitted{
		Amount:         twoMillionFTG,
		BuyerReference: "INSTITUTIONAL-BUYER-10M",
		PricePerToken:  math.LegacyNewDecWithPrec(5, 0), // $5.00
	}

	commitment, err := k.TransferMintingToCommitted(ctx, ownerAddr, commitReq)
	require.NoError(t, err)
	require.Equal(t, "funded", commitment.Status)

	// Verify minting wallet is now empty
	mintingBalance := bankKeeper.GetBalance(ctx, mintingAddr, "uftg")
	require.True(t, mintingBalance.Amount.IsZero())

	// Verify committed wallet holds the tokens
	committedBalance := bankKeeper.GetBalance(ctx, committedAddr, "uftg")
	require.Equal(t, twoMillionFTG, committedBalance.Amount)

	// Step 3: Deliver to buyer
	deliverReq := treasury.DeliverCommitment{
		CommitmentID:      commitment.CommitmentID,
		DestinationWallet: buyerAddr,
	}

	err = k.DeliverCommitment(ctx, ownerAddr, deliverReq)
	require.NoError(t, err)

	// Verify buyer has 2M FTG
	buyerBalance := bankKeeper.GetBalance(ctx, buyerAddr, "uftg")
	require.Equal(t, twoMillionFTG, buyerBalance.Amount)

	// Verify committed wallet is now empty
	committedBalance = bankKeeper.GetBalance(ctx, committedAddr, "uftg")
	require.True(t, committedBalance.Amount.IsZero())

	// Verify commitment is delivered
	final, found := k.GetCommitment(ctx, commitment.CommitmentID)
	require.True(t, found)
	require.Equal(t, "delivered", final.Status)
}
