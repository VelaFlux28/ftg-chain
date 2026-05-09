package treasury

import (
	"fmt"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module interface
type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

// Keeper of the treasury store
type Keeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	logger     log.Logger
	bankKeeper BankKeeper
	config     TreasuryConfig
}

// NewKeeper creates a new treasury Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
	bankKeeper BankKeeper,
	config TreasuryConfig,
) Keeper {
	return Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		logger:     logger,
		bankKeeper: bankKeeper,
		config:     config,
	}
}

// Logger returns a module-specific logger
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", ModuleName))
}

// GetMintingBalance returns the balance of the minting wallet
func (k Keeper) GetMintingBalance(ctx sdk.Context) sdk.Coin {
	return k.bankKeeper.GetBalance(ctx, k.config.MintingWalletAddress, "uftg")
}

// GetCommittedBalance returns the balance of the committed wallet
func (k Keeper) GetCommittedBalance(ctx sdk.Context) sdk.Coin {
	return k.bankKeeper.GetBalance(ctx, k.config.CommittedWalletAddress, "uftg")
}

// GetLiquidityBalance returns the balance of the liquidity wallet
func (k Keeper) GetLiquidityBalance(ctx sdk.Context) sdk.Coin {
	return k.bankKeeper.GetBalance(ctx, k.config.LiquidityWalletAddress, "uftg")
}

// TransferMintingToCommitted moves tokens from Minting → Committed wallet
// and creates a commitment record for the buyer
func (k Keeper) TransferMintingToCommitted(
	ctx sdk.Context,
	sender sdk.AccAddress,
	req TransferToCommitted,
) (*Commitment, error) {
	// Verify sender is the owner
	if !sender.Equals(k.config.OwnerAddress) {
		return nil, fmt.Errorf("unauthorized: only owner can transfer to committed")
	}

	// Check minting wallet has sufficient balance
	balance := k.GetMintingBalance(ctx)
	if balance.Amount.LT(req.Amount) {
		return nil, fmt.Errorf("insufficient minting balance: has %s, needs %s",
			balance.Amount.String(), req.Amount.String())
	}

	// Execute transfer
	coins := sdk.NewCoins(sdk.NewCoin("uftg", req.Amount))
	err := k.bankKeeper.SendCoins(ctx, k.config.MintingWalletAddress, k.config.CommittedWalletAddress, coins)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer to committed wallet: %w", err)
	}

	// Create commitment record
	commitment := &Commitment{
		CommitmentID:   k.GenerateCommitmentID(ctx),
		BuyerReference: req.BuyerReference,
		Amount:         req.Amount,
		PricePerToken:  req.PricePerToken,
		TotalValueUSD:  req.PricePerToken.MulInt(req.Amount).QuoInt(math.NewInt(1_000_000)), // Convert from uftg
		Status:         "funded",
		CreatedAt:      time.Now().UTC().Unix(),
	}

	k.SetCommitment(ctx, *commitment)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"treasury_commit",
			sdk.NewAttribute("commitment_id", commitment.CommitmentID),
			sdk.NewAttribute("buyer_reference", commitment.BuyerReference),
			sdk.NewAttribute("amount", commitment.Amount.String()),
			sdk.NewAttribute("price_per_token", commitment.PricePerToken.String()),
		),
	)

	k.Logger().Info("TransferMintingToCommitted completed",
		"commitment_id", commitment.CommitmentID,
		"amount", req.Amount.String(),
		"buyer", req.BuyerReference,
	)

	return commitment, nil
}

// TransferMintingToLiquidity moves tokens from Minting → Liquidity wallet
// making them available for public DEX trading
func (k Keeper) TransferMintingToLiquidity(
	ctx sdk.Context,
	sender sdk.AccAddress,
	req TransferToLiquidity,
) error {
	if !sender.Equals(k.config.OwnerAddress) {
		return fmt.Errorf("unauthorized: only owner can transfer to liquidity")
	}

	balance := k.GetMintingBalance(ctx)
	if balance.Amount.LT(req.Amount) {
		return fmt.Errorf("insufficient minting balance: has %s, needs %s",
			balance.Amount.String(), req.Amount.String())
	}

	coins := sdk.NewCoins(sdk.NewCoin("uftg", req.Amount))
	err := k.bankKeeper.SendCoins(ctx, k.config.MintingWalletAddress, k.config.LiquidityWalletAddress, coins)
	if err != nil {
		return fmt.Errorf("failed to transfer to liquidity wallet: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"treasury_to_liquidity",
			sdk.NewAttribute("amount", req.Amount.String()),
		),
	)

	k.Logger().Info("TransferMintingToLiquidity completed", "amount", req.Amount.String())
	return nil
}

// DeliverToCommitment delivers committed tokens to the buyer's wallet
func (k Keeper) DeliverCommitment(
	ctx sdk.Context,
	sender sdk.AccAddress,
	req DeliverCommitment,
) error {
	if !sender.Equals(k.config.OwnerAddress) {
		return fmt.Errorf("unauthorized: only owner can deliver commitments")
	}

	commitment, found := k.GetCommitment(ctx, req.CommitmentID)
	if !found {
		return fmt.Errorf("commitment %s not found", req.CommitmentID)
	}

	if commitment.Status != "funded" {
		return fmt.Errorf("commitment %s is not in funded status (current: %s)", req.CommitmentID, commitment.Status)
	}

	// Transfer from committed wallet to buyer
	coins := sdk.NewCoins(sdk.NewCoin("uftg", commitment.Amount))
	err := k.bankKeeper.SendCoins(ctx, k.config.CommittedWalletAddress, req.DestinationWallet, coins)
	if err != nil {
		return fmt.Errorf("failed to deliver to buyer: %w", err)
	}

	// Update commitment status
	commitment.Status = "delivered"
	commitment.DeliveredAt = time.Now().UTC().Unix()
	commitment.DestinationWallet = req.DestinationWallet
	k.SetCommitment(ctx, commitment)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"treasury_deliver",
			sdk.NewAttribute("commitment_id", commitment.CommitmentID),
			sdk.NewAttribute("buyer_reference", commitment.BuyerReference),
			sdk.NewAttribute("amount", commitment.Amount.String()),
			sdk.NewAttribute("destination", req.DestinationWallet.String()),
		),
	)

	k.Logger().Info("DeliverCommitment completed",
		"commitment_id", commitment.CommitmentID,
		"destination", req.DestinationWallet.String(),
	)

	return nil
}

// CancelCommitment returns committed tokens back to the minting wallet
func (k Keeper) CancelCommitmentOp(
	ctx sdk.Context,
	sender sdk.AccAddress,
	req CancelCommitment,
) error {
	if !sender.Equals(k.config.OwnerAddress) {
		return fmt.Errorf("unauthorized: only owner can cancel commitments")
	}

	commitment, found := k.GetCommitment(ctx, req.CommitmentID)
	if !found {
		return fmt.Errorf("commitment %s not found", req.CommitmentID)
	}

	if commitment.Status != "funded" && commitment.Status != "pending" {
		return fmt.Errorf("commitment %s cannot be cancelled (status: %s)", req.CommitmentID, commitment.Status)
	}

	// Return tokens to minting wallet
	coins := sdk.NewCoins(sdk.NewCoin("uftg", commitment.Amount))
	err := k.bankKeeper.SendCoins(ctx, k.config.CommittedWalletAddress, k.config.MintingWalletAddress, coins)
	if err != nil {
		return fmt.Errorf("failed to return tokens to minting wallet: %w", err)
	}

	commitment.Status = "cancelled"
	k.SetCommitment(ctx, commitment)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"treasury_cancel",
			sdk.NewAttribute("commitment_id", commitment.CommitmentID),
			sdk.NewAttribute("reason", req.Reason),
			sdk.NewAttribute("amount_returned", commitment.Amount.String()),
		),
	)

	return nil
}

// GetTreasuryOverview returns a summary of all treasury wallets
func (k Keeper) GetTreasuryOverview(ctx sdk.Context) map[string]string {
	return map[string]string{
		"minting_balance":   k.GetMintingBalance(ctx).Amount.String(),
		"committed_balance": k.GetCommittedBalance(ctx).Amount.String(),
		"liquidity_balance": k.GetLiquidityBalance(ctx).Amount.String(),
		"minting_address":   k.config.MintingWalletAddress.String(),
		"committed_address": k.config.CommittedWalletAddress.String(),
		"liquidity_address": k.config.LiquidityWalletAddress.String(),
	}
}

// Store helpers

func (k Keeper) SetCommitment(ctx sdk.Context, commitment Commitment) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&commitment)
	store.Set([]byte("Commitment/"+commitment.CommitmentID), bz)
}

func (k Keeper) GetCommitment(ctx sdk.Context, commitmentID string) (Commitment, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte("Commitment/" + commitmentID))
	if bz == nil {
		return Commitment{}, false
	}
	var commitment Commitment
	k.cdc.MustUnmarshal(bz, &commitment)
	return commitment, true
}

func (k Keeper) GenerateCommitmentID(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte("Commitment/count"))
	var count uint64
	if bz != nil {
		count = sdk.BigEndianToUint64(bz)
	}
	count++
	store.Set([]byte("Commitment/count"), sdk.Uint64ToBigEndian(count))
	return fmt.Sprintf("FTG-COMMIT-%06d", count)
}
