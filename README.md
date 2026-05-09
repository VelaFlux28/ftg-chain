# FTG Energy Chain

**The world's first physics-backed digital energy blockchain.**

Built on Cosmos SDK. Protected by US Patent 11,962,710.

## Overview

FTG Energy Chain is a sovereign Cosmos SDK blockchain where every token is backed by verified stored energy. Unlike speculative cryptocurrencies, FTG tokens derive their value from real-world energy assets — Renewable Energy Certificates (RECs) and Strategic Energy Vaults powered by NFluxion Power Cells.

**1 FTG = 10 kWh of verified stored energy = $5.00 USD**

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  FTG Energy Chain                     │
│                  (ftg-energy-1)                       │
│                                                       │
│  ┌─────────────┐  ┌──────────┐  ┌──────────────┐    │
│  │ x/energymint│  │ x/ftgburn│  │ x/provenance │    │
│  │             │  │          │  │              │    │
│  │ REC Upload  │  │ Merchant │  │ Certificate  │    │
│  │ → Mint FTG  │  │ Protocol │  │ Registry     │    │
│  │             │  │ → Burn   │  │ + IPFS       │    │
│  └─────────────┘  └──────────┘  └──────────────┘    │
│                                                       │
│  ┌─────────────────────────────────────────────┐     │
│  │              x/treasury                       │     │
│  │  Minting → Committed → Liquidity → Buyer     │     │
│  └─────────────────────────────────────────────┘     │
│                                                       │
│  Standard Cosmos: auth, bank, staking, gov, dist     │
│  Consensus: CometBFT (→ MCS-enforced at mainnet)    │
└─────────────────────────────────────────────────────┘
```

## Custom Modules

### x/energymint — Certificate-Backed Minting
- Upload verified RECs or vault energy certificates
- Each MWh mints exactly 100 FTG tokens
- IPFS hash required for certificate document storage
- Double-mint protection (unique external IDs)
- Authorized minter whitelist (treasury wallets only)

### x/ftgburn — Automatic Merchant Protocol
- Energy vault sells power to grid → receives USD
- USD enters Liquidity Smart Contract
- Contract buys FTG on DEX at market price
- Purchased FTG is burned permanently (sent to null address)
- Creates continuous deflationary pressure
- Also supports manual burns and energy redemption burns

### x/provenance — Certificate Registry
- Immutable on-chain record for every energy certificate
- Links tokens to specific energy sources (solar, wind, hydro, etc.)
- IPFS document storage for certificates, invoices, meter readings
- Full chain of custody tracking
- Any token holder can trace their token to the original energy source

### x/treasury — Segregated Wallet Management
- **Minting Wallet**: Receives all newly minted tokens
- **Committed Wallet**: Tokens reserved for OTC buyers (off-market)
- **Liquidity Wallet**: Tokens available for public DEX trading
- Commitment tracking for OTC deals
- Delivery and cancellation workflows

## Token Economics

| Property | Value |
|----------|-------|
| Symbol | FTG |
| Base Denom | uftg (1 FTG = 1,000,000 uftg) |
| Energy Backing | 10 kWh per token |
| Price Anchor | $5.00 USD |
| Inflation | 0% (zero — supply-controlled) |
| Genesis Supply | 61,700 FTG (617 MWh existing RECs) |

## Quick Start

```bash
# Build
make build

# Initialize node
make init

# Start testnet
make docker-build
make docker-run

# Check status
curl http://localhost:26657/status

# Treasury overview
./build/ftgd treasury overview

# Mint tokens from certificate
./build/ftgd mint from-certificate \
  --external-id "TP-2024-001" \
  --source "TerraPass" \
  --mwh 100 \
  --ipfs-hash "QmXxx..." \
  --provenance-type "proof_of_purchase"
```

## Ports

| Port | Service |
|------|---------|
| 26656 | P2P |
| 26657 | RPC |
| 1317 | REST API |
| 9090 | gRPC |
| 9091 | gRPC Web |
| 26660 | Prometheus |

## Patents

- **US 11,962,710** — Energy Block Chain (granted 2024)
- **17/863,165** — Gateway/Toll Booth claims (allowed April 2026)

## Regulatory

Per SEC Release 33-11412 (March 2026): FTG is classified as a **Digital Commodity** — not a security. The burn-only protocol ensures tokens fail the Howey test (no expectation of profit from others' efforts).

## License

Proprietary. Protected by US Patent 11,962,710.
Copyright © 2026 FTG Energy / NFluxion LLC. All rights reserved.
