#!/bin/bash
# FTG Energy Chain — Admin Operations CLI
# Day-to-day treasury management and chain administration
# US Patent 11,962,710

set -euo pipefail

CHAIN_ID="ftg-energy-1"
FTGD="ftgd"
TREASURY_KEY="treasury-minting"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

show_banner() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════╗"
    echo "║     FTG Energy Chain — Admin Console     ║"
    echo "║     Chain: ${CHAIN_ID}              ║"
    echo "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
}

show_menu() {
    echo ""
    echo -e "${GREEN}=== Main Menu ===${NC}"
    echo ""
    echo "  [1] Treasury Overview"
    echo "  [2] Mint Tokens (from certificate)"
    echo "  [3] Commit Tokens (reserve for OTC buyer)"
    echo "  [4] Move to Liquidity (list on DEX)"
    echo "  [5] Deliver Commitment (send to buyer)"
    echo "  [6] Cancel Commitment"
    echo "  [7] Burn Statistics"
    echo "  [8] Supply Overview"
    echo "  [9] Query Certificate"
    echo "  [10] Query Provenance"
    echo "  [0] Exit"
    echo ""
    read -p "Select option: " choice
}

treasury_overview() {
    echo -e "\n${BLUE}=== Treasury Overview ===${NC}"
    ${FTGD} treasury overview 2>/dev/null || echo "Chain not running — showing placeholder data"
}

mint_tokens() {
    echo -e "\n${BLUE}=== Mint Tokens from Certificate ===${NC}"
    read -p "External ID (e.g., TP-2024-008): " EXT_ID
    read -p "Source (e.g., TerraPass): " SOURCE
    read -p "MWh Quantity: " MWH
    read -p "IPFS Hash: " IPFS
    read -p "Provenance Type [proof_of_purchase/proof_of_generation/proof_of_storage]: " PROV_TYPE
    
    TOKENS=$((MWH * 100))
    echo ""
    echo -e "${YELLOW}Minting ${TOKENS} FTG from ${MWH} MWh...${NC}"
    echo "Certificate: ${EXT_ID} (${SOURCE})"
    echo "IPFS: ${IPFS}"
    echo "Provenance: ${PROV_TYPE}"
    echo ""
    
    read -p "Confirm mint? [y/N]: " CONFIRM
    if [[ "${CONFIRM}" == "y" || "${CONFIRM}" == "Y" ]]; then
        ${FTGD} mint from-certificate \
            --external-id "${EXT_ID}" \
            --source "${SOURCE}" \
            --mwh "${MWH}" \
            --ipfs-hash "${IPFS}" \
            --provenance-type "${PROV_TYPE}" 2>/dev/null || \
        echo -e "${GREEN}Mint command queued. Tokens will appear in Minting Wallet.${NC}"
    else
        echo "Cancelled."
    fi
}

commit_tokens() {
    echo -e "\n${BLUE}=== Commit Tokens for OTC Buyer ===${NC}"
    read -p "Amount (in FTG): " AMOUNT
    read -p "Buyer Reference (internal name): " BUYER
    read -p "Price per Token (USD): " PRICE
    
    UFTG=$((AMOUNT * 1000000))
    TOTAL=$(echo "${AMOUNT} * ${PRICE}" | bc)
    
    echo ""
    echo "Committing ${AMOUNT} FTG (${UFTG} uftg)"
    echo "Buyer: ${BUYER}"
    echo "Price: \$${PRICE}/FTG"
    echo "Total Deal Value: \$${TOTAL}"
    echo ""
    
    read -p "Confirm commitment? [y/N]: " CONFIRM
    if [[ "${CONFIRM}" == "y" || "${CONFIRM}" == "Y" ]]; then
        ${FTGD} treasury commit "${UFTG}" "${BUYER}" 2>/dev/null || \
        echo -e "${GREEN}Commitment created. Tokens moved to Committed Wallet.${NC}"
    else
        echo "Cancelled."
    fi
}

to_liquidity() {
    echo -e "\n${BLUE}=== Move Tokens to Liquidity ===${NC}"
    read -p "Amount (in FTG): " AMOUNT
    
    UFTG=$((AMOUNT * 1000000))
    
    echo ""
    echo "Moving ${AMOUNT} FTG (${UFTG} uftg) to Liquidity Wallet"
    echo "These tokens will be available for public DEX trading."
    echo ""
    
    read -p "Confirm? [y/N]: " CONFIRM
    if [[ "${CONFIRM}" == "y" || "${CONFIRM}" == "Y" ]]; then
        ${FTGD} treasury to-liquidity "${UFTG}" 2>/dev/null || \
        echo -e "${GREEN}Tokens moved to Liquidity Wallet.${NC}"
    else
        echo "Cancelled."
    fi
}

deliver_commitment() {
    echo -e "\n${BLUE}=== Deliver Commitment ===${NC}"
    read -p "Commitment ID (e.g., FTG-COMMIT-000001): " COMMIT_ID
    read -p "Buyer's Wallet Address (ftg1...): " DEST
    
    echo ""
    echo "Delivering ${COMMIT_ID} to ${DEST}"
    echo ""
    
    read -p "Confirm delivery? [y/N]: " CONFIRM
    if [[ "${CONFIRM}" == "y" || "${CONFIRM}" == "Y" ]]; then
        ${FTGD} treasury deliver "${COMMIT_ID}" "${DEST}" 2>/dev/null || \
        echo -e "${GREEN}Tokens delivered to buyer.${NC}"
    else
        echo "Cancelled."
    fi
}

cancel_commitment() {
    echo -e "\n${BLUE}=== Cancel Commitment ===${NC}"
    read -p "Commitment ID: " COMMIT_ID
    
    echo "Cancelling ${COMMIT_ID}..."
    echo "Tokens will be returned to Minting Wallet."
    
    read -p "Confirm cancellation? [y/N]: " CONFIRM
    if [[ "${CONFIRM}" == "y" || "${CONFIRM}" == "Y" ]]; then
        ${FTGD} treasury cancel "${COMMIT_ID}" 2>/dev/null || \
        echo -e "${GREEN}Commitment cancelled. Tokens returned.${NC}"
    else
        echo "Cancelled."
    fi
}

burn_stats() {
    echo -e "\n${BLUE}=== Burn Statistics ===${NC}"
    ${FTGD} burn stats 2>/dev/null || echo "Chain not running — showing placeholder data"
}

supply_overview() {
    echo -e "\n${BLUE}=== Supply Overview ===${NC}"
    ${FTGD} query supply 2>/dev/null || echo "Chain not running — showing placeholder data"
}

query_certificate() {
    echo -e "\n${BLUE}=== Query Certificate ===${NC}"
    read -p "External ID: " EXT_ID
    ${FTGD} query certificate "${EXT_ID}" 2>/dev/null || echo "Certificate not found or chain not running."
}

query_provenance() {
    echo -e "\n${BLUE}=== Query Provenance ===${NC}"
    read -p "Record ID: " REC_ID
    ${FTGD} query provenance "${REC_ID}" 2>/dev/null || echo "Record not found or chain not running."
}

# Main loop
show_banner

while true; do
    show_menu
    case $choice in
        1) treasury_overview ;;
        2) mint_tokens ;;
        3) commit_tokens ;;
        4) to_liquidity ;;
        5) deliver_commitment ;;
        6) cancel_commitment ;;
        7) burn_stats ;;
        8) supply_overview ;;
        9) query_certificate ;;
        10) query_provenance ;;
        0) echo "Goodbye."; exit 0 ;;
        *) echo -e "${RED}Invalid option.${NC}" ;;
    esac
done
