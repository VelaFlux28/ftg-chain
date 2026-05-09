package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/VelaFlux28/ftg-chain/x/energymint/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the energymint MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// MintFromCertificate handles the MsgMintFromCertificate transaction
func (m msgServer) MintFromCertificate(goCtx context.Context, msg *types.MsgMintFromCertificate) (*types.MsgMintFromCertificateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, fmt.Errorf("invalid sender address: %w", err)
	}

	var destWallet sdk.AccAddress
	if msg.DestinationWallet != "" {
		destWallet, err = sdk.AccAddressFromBech32(msg.DestinationWallet)
		if err != nil {
			return nil, fmt.Errorf("invalid destination wallet: %w", err)
		}
	}

	cert := types.Certificate{
		ExternalID:     msg.ExternalId,
		Source:         msg.Source,
		MwhQuantity:    msg.MwhQuantity,
		GenerationDate: msg.GenerationDate,
		PurchaseDate:   msg.PurchaseDate,
		IPFSHash:       msg.IpfsHash,
		ProvenanceType: msg.ProvenanceType,
	}

	resp, err := m.Keeper.MintFromCertificate(ctx, sender, cert, destWallet)
	if err != nil {
		return nil, err
	}

	return &types.MsgMintFromCertificateResponse{
		CertificateId: resp.CertificateID,
		TokensMinted:  resp.TokensMinted.String(),
		Recipient:     resp.Recipient.String(),
	}, nil
}

// AddAuthorizedMinter handles adding a new authorized minter address
func (m msgServer) AddMinter(goCtx context.Context, msg *types.MsgAddMinter) (*types.MsgAddMinterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Only the module authority (governance) can add minters
	if msg.Authority != m.authority {
		return nil, fmt.Errorf("unauthorized: only governance can add minters")
	}

	minterAddr, err := sdk.AccAddressFromBech32(msg.MinterAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid minter address: %w", err)
	}

	m.Keeper.AddAuthorizedMinter(ctx, minterAddr)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"minter_added",
			sdk.NewAttribute("minter_address", msg.MinterAddress),
			sdk.NewAttribute("added_by", msg.Authority),
		),
	)

	return &types.MsgAddMinterResponse{}, nil
}

// RemoveMinter handles removing an authorized minter address
func (m msgServer) RemoveMinter(goCtx context.Context, msg *types.MsgRemoveMinter) (*types.MsgRemoveMinterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.Authority != m.authority {
		return nil, fmt.Errorf("unauthorized: only governance can remove minters")
	}

	minterAddr, err := sdk.AccAddressFromBech32(msg.MinterAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid minter address: %w", err)
	}

	m.Keeper.RemoveAuthorizedMinter(ctx, minterAddr)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"minter_removed",
			sdk.NewAttribute("minter_address", msg.MinterAddress),
			sdk.NewAttribute("removed_by", msg.Authority),
		),
	)

	return &types.MsgRemoveMinterResponse{}, nil
}
