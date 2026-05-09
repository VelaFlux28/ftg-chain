#!/bin/bash
# FTG Energy Chain — Testnet Deployment Script
# Deploys to EC2 instance alongside existing VELA infrastructure
# US Patent 11,962,710

set -euo pipefail

# Configuration
DEPLOY_DIR="/opt/ftg-chain"
CHAIN_ID="ftg-energy-1"
MONIKER="ftg-genesis-validator"
LOG_FILE="/var/log/ftg-chain.log"

echo "╔══════════════════════════════════════════╗"
echo "║     FTG Energy Chain — Testnet Deploy    ║"
echo "║     Chain ID: ${CHAIN_ID}           ║"
echo "║     Patent: US 11,962,710                ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# Check if running as root or with sudo
if [ "$EUID" -ne 0 ]; then
    echo "Please run with sudo"
    exit 1
fi

# Step 1: Install Go if not present
if ! command -v go &> /dev/null; then
    echo "[1/7] Installing Go 1.22..."
    wget -q https://go.dev/dl/go1.22.4.linux-amd64.tar.gz -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin
    echo "Go $(go version) installed."
else
    echo "[1/7] Go already installed: $(go version)"
fi

# Step 2: Create deployment directory
echo "[2/7] Setting up deployment directory..."
mkdir -p ${DEPLOY_DIR}
cd ${DEPLOY_DIR}

# Step 3: Copy chain code (assumes git clone or scp already done)
echo "[3/7] Building ftgd binary..."
if [ -f "go.mod" ]; then
    go mod tidy
    go build -o /usr/local/bin/ftgd ./cmd/ftgd/
    echo "ftgd binary built and installed."
else
    echo "ERROR: Chain code not found in ${DEPLOY_DIR}"
    echo "Please clone or copy the ftg-chain repository first."
    exit 1
fi

# Step 4: Initialize node
echo "[4/7] Initializing node..."
FTGD_HOME="/root/.ftgd"
mkdir -p ${FTGD_HOME}/config
cp genesis/genesis.json ${FTGD_HOME}/config/genesis.json
cp config/config.toml ${FTGD_HOME}/config/config.toml
cp config/app.toml ${FTGD_HOME}/config/app.toml
echo "Node initialized at ${FTGD_HOME}"

# Step 5: Create systemd service
echo "[5/7] Creating systemd service..."
cat > /etc/systemd/system/ftg-chain.service << EOF
[Unit]
Description=FTG Energy Chain Node
Documentation=https://ftg.energy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/ftgd start
Restart=always
RestartSec=3
LimitNOFILE=65535
StandardOutput=append:${LOG_FILE}
StandardError=append:${LOG_FILE}
Environment="PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
echo "Systemd service created."

# Step 6: Configure firewall (if ufw is active)
echo "[6/7] Configuring firewall..."
if command -v ufw &> /dev/null && ufw status | grep -q "active"; then
    ufw allow 26656/tcp comment "FTG P2P"
    ufw allow 26657/tcp comment "FTG RPC"
    ufw allow 1317/tcp comment "FTG REST API"
    ufw allow 9090/tcp comment "FTG gRPC"
    echo "Firewall rules added."
else
    echo "UFW not active, skipping firewall config."
fi

# Step 7: Start the node
echo "[7/7] Starting FTG Energy Chain..."
systemctl enable ftg-chain
systemctl start ftg-chain

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║     FTG Energy Chain — DEPLOYED          ║"
echo "╠══════════════════════════════════════════╣"
echo "║  Status: systemctl status ftg-chain      ║"
echo "║  Logs:   tail -f ${LOG_FILE}  ║"
echo "║  RPC:    http://localhost:26657          ║"
echo "║  API:    http://localhost:1317           ║"
echo "║  gRPC:   localhost:9090                  ║"
echo "╚══════════════════════════════════════════╝"
