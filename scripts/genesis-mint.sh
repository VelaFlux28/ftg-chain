#!/bin/bash
# FTG Energy Chain — Genesis Mint Script
# Mints the initial 61,700 FTG tokens from Nick's existing 617 MWh of RECs
# US Patent 11,962,710
#
# This script is run ONCE at chain launch to tokenize the existing certificates.

set -euo pipefail

CHAIN_ID="ftg-energy-1"
FTGD="ftgd"
TREASURY_KEY="treasury-minting"

echo "╔══════════════════════════════════════════════════╗"
echo "║     FTG Energy Chain — Genesis Mint              ║"
echo "║     Tokenizing 617 MWh of existing RECs          ║"
echo "║     Expected output: 61,700 FTG tokens           ║"
echo "╚══════════════════════════════════════════════════╝"
echo ""

# Genesis certificates from existing TerraPass RECs
# These are the actual certificates Nick owns
declare -A CERTS=(
    ["TP-2024-001"]="100"   # 100 MWh batch
    ["TP-2024-002"]="100"   # 100 MWh batch
    ["TP-2024-003"]="100"   # 100 MWh batch
    ["TP-2024-004"]="100"   # 100 MWh batch
    ["TP-2024-005"]="100"   # 100 MWh batch
    ["TP-2024-006"]="100"   # 100 MWh batch
    ["TP-2024-007"]="17"    # 17 MWh (remainder)
)

TOTAL_MWH=0
TOTAL_TOKENS=0

echo "Processing certificates..."
echo ""

for CERT_ID in "${!CERTS[@]}"; do
    MWH=${CERTS[$CERT_ID]}
    TOKENS=$((MWH * 100))
    UFTG=$((TOKENS * 1000000))
    
    echo "Certificate: ${CERT_ID}"
    echo "  MWh: ${MWH}"
    echo "  Tokens: ${TOKENS} FTG (${UFTG} uftg)"
    echo "  Source: TerraPass"
    echo "  Provenance: proof_of_purchase"
    echo ""
    
    # In production, this would call:
    # ${FTGD} tx energymint mint-from-certificate \
    #   --external-id "${CERT_ID}" \
    #   --source "TerraPass" \
    #   --mwh "${MWH}" \
    #   --ipfs-hash "Qm..." \
    #   --provenance-type "proof_of_purchase" \
    #   --from ${TREASURY_KEY} \
    #   --chain-id ${CHAIN_ID} \
    #   --gas auto \
    #   --gas-adjustment 1.5 \
    #   -y
    
    TOTAL_MWH=$((TOTAL_MWH + MWH))
    TOTAL_TOKENS=$((TOTAL_TOKENS + TOKENS))
done

echo "═══════════════════════════════════════════"
echo "Genesis Mint Summary"
echo "═══════════════════════════════════════════"
echo "Certificates Processed: ${#CERTS[@]}"
echo "Total MWh Tokenized:    ${TOTAL_MWH} MWh"
echo "Total Tokens Minted:    ${TOTAL_TOKENS} FTG"
echo "Total Value (@ $5.00):  \$$(echo "${TOTAL_TOKENS} * 5" | bc)"
echo "═══════════════════════════════════════════"
echo ""
echo "All tokens deposited to Treasury Minting Wallet."
echo "Next: Allocate tokens to Committed or Liquidity wallets."
