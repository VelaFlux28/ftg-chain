package types

import (
	"context"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TypeMsgMintFromCertificate = "mint_from_certificate"
	TypeMsgAddMinter           = "add_minter"
	TypeMsgRemoveMinter        = "remove_minter"
)

type MsgMintFromCertificate struct {
	Sender            string         `json:"sender"`
	ExternalId        string         `json:"external_id"`
	Source            string         `json:"source"`
	MwhQuantity       math.LegacyDec `json:"mwh_quantity"`
	GenerationDate    time.Time      `json:"generation_date"`
	PurchaseDate      time.Time      `json:"purchase_date"`
	IpfsHash          string         `json:"ipfs_hash"`
	ProvenanceType    string         `json:"provenance_type"`
	DestinationWallet string         `json:"destination_wallet,omitempty"`
}

type MsgMintFromCertificateResponse struct {
	CertificateId string `json:"certificate_id"`
	TokensMinted  string `json:"tokens_minted"`
	Recipient     string `json:"recipient"`
}

type MsgAddMinter struct {
	Authority     string `json:"authority"`
	MinterAddress string `json:"minter_address"`
}

type MsgAddMinterResponse struct{}

type MsgRemoveMinter struct {
	Authority     string `json:"authority"`
	MinterAddress string `json:"minter_address"`
}

type MsgRemoveMinterResponse struct{}

// MsgServer defines the energymint module's gRPC message service
type MsgServer interface {
	MintFromCertificate(ctx context.Context, msg *MsgMintFromCertificate) (*MsgMintFromCertificateResponse, error)
	AddMinter(ctx context.Context, msg *MsgAddMinter) (*MsgAddMinterResponse, error)
	RemoveMinter(ctx context.Context, msg *MsgRemoveMinter) (*MsgRemoveMinterResponse, error)
}

func (msg *MsgMintFromCertificate) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return err
	}

	if msg.ExternalId == "" {
		return ErrInvalidCertificate("external_id cannot be empty")
	}

	if msg.Source == "" {
		return ErrInvalidCertificate("source cannot be empty")
	}

	if msg.MwhQuantity.IsNil() || msg.MwhQuantity.IsNegative() || msg.MwhQuantity.IsZero() {
		return ErrInvalidCertificate("mwh_quantity must be positive")
	}

	if msg.IpfsHash == "" {
		return ErrInvalidCertificate("ipfs_hash cannot be empty")
	}

	if msg.ProvenanceType == "" {
		return ErrInvalidCertificate("provenance_type cannot be empty")
	}

	if msg.DestinationWallet != "" {
		_, err := sdk.AccAddressFromBech32(msg.DestinationWallet)
		if err != nil {
			return err
		}
	}

	return nil
}

func (msg *MsgMintFromCertificate) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}
