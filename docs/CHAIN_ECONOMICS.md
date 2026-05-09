# FTG Energy Chain — Token Economics & Chain Constitution

## Chain Identity

| Parameter | Value |
|-----------|-------|
| Chain ID | `ftg-energy-1` |
| Token Name | FTG Energy Token |
| Symbol | FTG |
| Base Denom | `uftg` (micro-FTG) |
| Display Denom | `ftg` |
| Decimals | 6 (1 FTG = 1,000,000 uftg) |

## Energy Backing

| Parameter | Value |
|-----------|-------|
| Energy per Token | 10 kWh |
| Tokens per MWh | 100 FTG |
| Price Anchor | $5.00 USD per FTG |
| Price Derivation | $25,000 NFluxion Power Cell ÷ 5,000 tokens |

## Price Breakdown ($5.00 per FTG)

| Component | Amount | Purpose |
|-----------|--------|---------|
| Wholesale Electricity | $1.45 | Raw energy procurement cost |
| Verification/ESG | $0.80 | Certificate verification, ESG compliance |
| Storage/Vault | $1.00 | Physical storage infrastructure |
| IP Licensing | $1.25 | Patent royalty (US 11,962,710) |
| Liquidity/Burn Premium | $0.50 | DEX liquidity + deflationary premium |
| **Total** | **$5.00** | |

## Supply Model

**ZERO INFLATION.** FTG has no inflationary minting schedule.

- Standard Cosmos SDK `x/mint` module is set to 0% inflation permanently
- New tokens are ONLY created through the `x/energymint` module
- Every new token requires a verified energy certificate (REC or vault)
- Supply grows only when new energy backing is acquired
- Supply shrinks through the burn protocol

## Genesis Supply

| Source | MWh | Tokens | USD Value |
|--------|-----|--------|-----------|
| Existing TerraPass RECs | 617 | 61,700 FTG | $308,500 |

## Minting Rules (x/energymint)

1. Only authorized minter addresses can mint (Treasury wallets)
2. Every mint requires an uploaded certificate with IPFS hash
3. Minimum 1 MWh per certificate upload
4. Certificate external IDs are unique — no double-minting
5. Accepted provenance types: `proof_of_purchase`, `proof_of_generation`, `proof_of_storage`

## Burn Protocol (x/ftgburn)

The **Automatic Merchant Protocol** is the core deflationary engine:

1. Strategic Energy Vault sells energy to the grid
2. Grid pays rent in USD
3. USD enters the Liquidity Smart Contract
4. Contract buys FTG on the DEX at market price
5. Purchased FTG is burned to null address (permanently destroyed)

Additional burn sources:
- **Manual burn**: Any token holder can burn their tokens voluntarily
- **Redemption burn**: Token redeemed for physical energy delivery

## Validator Requirements

**Patent-Enforced Consensus (Future)**

In the final architecture, only devices running the patented MCS (Metering, Communication, and Storage) system can validate blocks. This creates a hardware-enforced validator set that cannot be replicated without patent licensing.

**Testnet Phase**: Standard CometBFT validators (ed25519)
**Mainnet Phase**: MCS device attestation required for validator registration

## Governance

| Parameter | Value |
|-----------|-------|
| Min Deposit | 10,000 FTG |
| Deposit Period | 48 hours |
| Voting Period | 7 days |
| Quorum | 33.4% |
| Pass Threshold | 50% |
| Veto Threshold | 33.4% |

## Staking

| Parameter | Value |
|-----------|-------|
| Unbonding Period | 21 days |
| Max Validators | 100 |
| Bond Denom | uftg |
| Min Commission | 5% |

## IBC Bridges (Planned)

| Bridge | Purpose |
|--------|---------|
| Osmosis | DEX liquidity, public trading |
| Noble | USDC settlement for purchases |
| Ethereum (future) | Wrapped FTG for DeFi |
| Solana (future) | High-speed trading |

## Patents

- **US 11,962,710** — Energy Block Chain (granted 2024)
- **17/863,165** — Gateway/Toll Booth claims (allowed April 2026)

## Regulatory Classification

Per SEC Release 33-11412 (March 2026): FTG is classified as a **Digital Commodity** — NOT a security. No securities registration required.

## Key Addresses (Testnet)

| Wallet | Purpose |
|--------|---------|
| Treasury (Minting) | Receives all newly minted tokens |
| Treasury (Committed) | Holds tokens earmarked for OTC buyers |
| Treasury (Liquidity) | Tokens listed on DEX for public purchase |
| Burn Module | Receives and destroys burned tokens |
| Liquidity Contract | Executes Automatic Merchant Protocol |
