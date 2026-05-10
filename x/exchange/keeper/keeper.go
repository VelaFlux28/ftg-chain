package keeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/VelaFlux28/ftg-chain/x/exchange/types"
)

// Keeper manages the exchange state with persistent JSON storage
type Keeper struct {
	mu sync.RWMutex

	// State
	Wallets     map[string]*types.ExchangeWallet `json:"wallets"`
	Orders      map[string]*types.Order          `json:"orders"`
	Trades      []*types.Trade                   `json:"trades"`
	KYCApps     map[string]*types.KYCApplication `json:"kyc_applications"`
	Pairs       map[string]*types.TradingPair    `json:"pairs"`
	DailyLimits map[string]*types.DailyLimit     `json:"daily_limits"`
	Config      *types.ExchangeConfig            `json:"config"`

	// Order book: pairID -> side -> sorted orders
	BuyBook  map[string][]*types.Order `json:"-"`
	SellBook map[string][]*types.Order `json:"-"`

	// Sequences
	WalletSeq int64 `json:"wallet_seq"`
	OrderSeq  int64 `json:"order_seq"`
	TradeSeq  int64 `json:"trade_seq"`
	KYCSeq    int64 `json:"kyc_seq"`

	// Persistence
	filePath string
}

// NewKeeper creates or loads the exchange keeper
func NewKeeper(homeDir string) *Keeper {
	statePath := filepath.Join(homeDir, "data", "exchange_state.json")

	k := &Keeper{
		Wallets:     make(map[string]*types.ExchangeWallet),
		Orders:      make(map[string]*types.Order),
		Trades:      make([]*types.Trade, 0),
		KYCApps:     make(map[string]*types.KYCApplication),
		Pairs:       make(map[string]*types.TradingPair),
		DailyLimits: make(map[string]*types.DailyLimit),
		BuyBook:     make(map[string][]*types.Order),
		SellBook:    make(map[string][]*types.Order),
		Config:      defaultConfig(),
		filePath:    statePath,
	}

	// Load existing state
	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, k)
	}

	// Initialize default trading pair if none exist
	if len(k.Pairs) == 0 {
		k.initDefaultPairs()
	}

	// Rebuild order books from persisted orders
	k.rebuildOrderBooks()

	return k
}

func defaultConfig() *types.ExchangeConfig {
	return &types.ExchangeConfig{
		FeeCollector:     "ftg_fee_collector",
		TradingEnabled:   true,
		RegistrationOpen: true,
		AMLThreshold:     10_000_000_000,
		TierLimits: map[types.KYCTier]int64{
			types.KYCTierNone:  0,
			types.KYCTierBasic: 1_000_000_000,
			types.KYCTierFull:  50_000_000_000,
			types.KYCTierInst:  0,
		},
	}
}

func (k *Keeper) initDefaultPairs() {
	k.Pairs["FTG-USDC"] = &types.TradingPair{
		ID:             "FTG-USDC",
		BaseDenom:      "uftg",
		QuoteDenom:     "uusdc",
		BaseSymbol:     "FTG",
		QuoteSymbol:    "USDC",
		MinOrderSize:   1_000_000,
		PriceIncrement: 10_000,
		SizeIncrement:  100_000,
		MakerFee:       10,
		TakerFee:       25,
		Active:         true,
	}
}

func (k *Keeper) rebuildOrderBooks() {
	k.BuyBook = make(map[string][]*types.Order)
	k.SellBook = make(map[string][]*types.Order)

	for _, order := range k.Orders {
		if order.Status == types.OrderStatusOpen || order.Status == types.OrderStatusPartial {
			if order.Side == types.OrderSideBuy {
				k.BuyBook[order.PairID] = append(k.BuyBook[order.PairID], order)
			} else {
				k.SellBook[order.PairID] = append(k.SellBook[order.PairID], order)
			}
		}
	}

	for pairID := range k.BuyBook {
		sort.Slice(k.BuyBook[pairID], func(i, j int) bool {
			return k.BuyBook[pairID][i].Price > k.BuyBook[pairID][j].Price
		})
	}
	for pairID := range k.SellBook {
		sort.Slice(k.SellBook[pairID], func(i, j int) bool {
			return k.SellBook[pairID][i].Price < k.SellBook[pairID][j].Price
		})
	}
}

// CreateWallet creates a new exchange wallet for a user
func (k *Keeper) CreateWallet(userID, label string) (*types.ExchangeWallet, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.Config.RegistrationOpen {
		return nil, types.ErrRegistrationClosed
	}

	for _, w := range k.Wallets {
		if w.UserID == userID && w.Active {
			return nil, types.ErrWalletExists
		}
	}

	k.WalletSeq++
	walletID := fmt.Sprintf("FTG-W-%06d", k.WalletSeq)
	address := fmt.Sprintf("ftg1exchange%s", walletID)

	wallet := &types.ExchangeWallet{
		ID:        walletID,
		UserID:    userID,
		Address:   address,
		Label:     label,
		CreatedAt: time.Now().UTC(),
		KYCTier:   types.KYCTierNone,
		Balances:  make(map[string]int64),
		Locked:    make(map[string]int64),
		Active:    true,
	}

	k.Wallets[walletID] = wallet
	k.save()
	return wallet, nil
}

// GetWallet returns a wallet by ID
func (k *Keeper) GetWallet(walletID string) (*types.ExchangeWallet, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	wallet, ok := k.Wallets[walletID]
	if !ok {
		return nil, types.ErrWalletNotFound
	}
	return wallet, nil
}

// GetWalletByUser returns a user's active wallet
func (k *Keeper) GetWalletByUser(userID string) (*types.ExchangeWallet, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	for _, w := range k.Wallets {
		if w.UserID == userID && w.Active {
			return w, nil
		}
	}
	return nil, types.ErrWalletNotFound
}

// DepositToWallet adds funds to a wallet
func (k *Keeper) DepositToWallet(walletID, denom string, amount int64) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	wallet, ok := k.Wallets[walletID]
	if !ok {
		return types.ErrWalletNotFound
	}
	if !wallet.Active {
		return types.ErrWalletInactive
	}

	wallet.Balances[denom] += amount
	k.save()
	return nil
}

// WithdrawFromWallet removes funds from a wallet
func (k *Keeper) WithdrawFromWallet(walletID, denom string, amount int64) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	wallet, ok := k.Wallets[walletID]
	if !ok {
		return types.ErrWalletNotFound
	}
	if !wallet.Active {
		return types.ErrWalletInactive
	}

	available := wallet.Balances[denom] - wallet.Locked[denom]
	if amount > available {
		return types.ErrInsufficientBalance
	}

	wallet.Balances[denom] -= amount
	k.save()
	return nil
}

// PlaceOrder places a new order on the book
func (k *Keeper) PlaceOrder(walletID, pairID string, side types.OrderSide, orderType types.OrderType, price, quantity int64, blockHeight int64) (*types.Order, []*types.Trade, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	wallet, ok := k.Wallets[walletID]
	if !ok {
		return nil, nil, types.ErrWalletNotFound
	}
	if !wallet.Active {
		return nil, nil, types.ErrWalletInactive
	}
	if wallet.KYCTier < types.KYCTierBasic {
		return nil, nil, types.ErrInsufficientKYC
	}
	if !k.Config.TradingEnabled {
		return nil, nil, types.ErrTradingDisabled
	}

	pair, ok := k.Pairs[pairID]
	if !ok {
		return nil, nil, types.ErrPairNotFound
	}
	if !pair.Active {
		return nil, nil, types.ErrPairInactive
	}
	if quantity < pair.MinOrderSize {
		return nil, nil, types.ErrBelowMinOrder
	}
	if orderType == types.OrderTypeLimit && price <= 0 {
		return nil, nil, types.ErrInvalidPrice
	}

	if err := k.checkDailyLimit(wallet, price, quantity); err != nil {
		return nil, nil, err
	}

	// Lock funds
	if side == types.OrderSideBuy {
		lockAmount := (price * quantity) / 1_000_000
		available := wallet.Balances[pair.QuoteDenom] - wallet.Locked[pair.QuoteDenom]
		if lockAmount > available {
			return nil, nil, types.ErrInsufficientBalance
		}
		wallet.Locked[pair.QuoteDenom] += lockAmount
	} else {
		available := wallet.Balances[pair.BaseDenom] - wallet.Locked[pair.BaseDenom]
		if quantity > available {
			return nil, nil, types.ErrInsufficientBalance
		}
		wallet.Locked[pair.BaseDenom] += quantity
	}

	k.OrderSeq++
	order := &types.Order{
		ID:           fmt.Sprintf("FTG-ORD-%08d", k.OrderSeq),
		WalletID:     walletID,
		UserID:       wallet.UserID,
		PairID:       pairID,
		Side:         side,
		Type:         orderType,
		Price:        price,
		Quantity:     quantity,
		FilledQty:    0,
		RemainingQty: quantity,
		Status:       types.OrderStatusOpen,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	k.Orders[order.ID] = order

	// Match
	trades := k.matchOrder(order, pair, blockHeight)

	// Add remainder to book
	if order.RemainingQty > 0 && orderType == types.OrderTypeLimit {
		if side == types.OrderSideBuy {
			k.BuyBook[pairID] = append(k.BuyBook[pairID], order)
			sort.Slice(k.BuyBook[pairID], func(i, j int) bool {
				return k.BuyBook[pairID][i].Price > k.BuyBook[pairID][j].Price
			})
		} else {
			k.SellBook[pairID] = append(k.SellBook[pairID], order)
			sort.Slice(k.SellBook[pairID], func(i, j int) bool {
				return k.SellBook[pairID][i].Price < k.SellBook[pairID][j].Price
			})
		}
	}

	k.save()
	return order, trades, nil
}

func (k *Keeper) matchOrder(order *types.Order, pair *types.TradingPair, blockHeight int64) []*types.Trade {
	var trades []*types.Trade
	var oppositeBook *[]*types.Order

	if order.Side == types.OrderSideBuy {
		book := k.SellBook[order.PairID]
		oppositeBook = &book
	} else {
		book := k.BuyBook[order.PairID]
		oppositeBook = &book
	}

	i := 0
	for i < len(*oppositeBook) && order.RemainingQty > 0 {
		maker := (*oppositeBook)[i]

		if order.Side == types.OrderSideBuy {
			if order.Type == types.OrderTypeLimit && maker.Price > order.Price {
				break
			}
		} else {
			if order.Type == types.OrderTypeLimit && maker.Price < order.Price {
				break
			}
		}

		if maker.WalletID == order.WalletID {
			i++
			continue
		}

		fillQty := order.RemainingQty
		if maker.RemainingQty < fillQty {
			fillQty = maker.RemainingQty
		}

		executionPrice := maker.Price
		makerFee := (fillQty * executionPrice * pair.MakerFee) / (1_000_000 * 10_000)
		takerFee := (fillQty * executionPrice * pair.TakerFee) / (1_000_000 * 10_000)

		k.TradeSeq++
		trade := &types.Trade{
			ID:          fmt.Sprintf("FTG-TRD-%08d", k.TradeSeq),
			PairID:      order.PairID,
			MakerOrder:  maker.ID,
			TakerOrder:  order.ID,
			MakerWallet: maker.WalletID,
			TakerWallet: order.WalletID,
			Side:        order.Side,
			Price:       executionPrice,
			Quantity:    fillQty,
			MakerFee:    makerFee,
			TakerFee:    takerFee,
			ExecutedAt:  time.Now().UTC(),
			BlockHeight: blockHeight,
		}
		trades = append(trades, trade)
		k.Trades = append(k.Trades, trade)

		k.settleTrade(trade, pair)

		order.FilledQty += fillQty
		order.RemainingQty -= fillQty
		maker.FilledQty += fillQty
		maker.RemainingQty -= fillQty

		if maker.RemainingQty == 0 {
			maker.Status = types.OrderStatusFilled
			*oppositeBook = append((*oppositeBook)[:i], (*oppositeBook)[i+1:]...)
		} else {
			maker.Status = types.OrderStatusPartial
			i++
		}
		maker.UpdatedAt = time.Now().UTC()
	}

	if order.RemainingQty == 0 {
		order.Status = types.OrderStatusFilled
	} else if order.FilledQty > 0 {
		order.Status = types.OrderStatusPartial
	}
	order.UpdatedAt = time.Now().UTC()

	if order.Side == types.OrderSideBuy {
		k.SellBook[order.PairID] = *oppositeBook
	} else {
		k.BuyBook[order.PairID] = *oppositeBook
	}

	return trades
}

func (k *Keeper) settleTrade(trade *types.Trade, pair *types.TradingPair) {
	makerWallet := k.Wallets[trade.MakerWallet]
	takerWallet := k.Wallets[trade.TakerWallet]
	quoteAmount := (trade.Price * trade.Quantity) / 1_000_000

	if trade.Side == types.OrderSideBuy {
		makerWallet.Locked[pair.BaseDenom] -= trade.Quantity
		makerWallet.Balances[pair.BaseDenom] -= trade.Quantity
		makerWallet.Balances[pair.QuoteDenom] += quoteAmount - trade.MakerFee

		takerWallet.Locked[pair.QuoteDenom] -= quoteAmount
		takerWallet.Balances[pair.QuoteDenom] -= quoteAmount
		takerWallet.Balances[pair.BaseDenom] += trade.Quantity
		takerWallet.Balances[pair.QuoteDenom] -= trade.TakerFee
	} else {
		makerWallet.Locked[pair.QuoteDenom] -= quoteAmount
		makerWallet.Balances[pair.QuoteDenom] -= quoteAmount
		makerWallet.Balances[pair.BaseDenom] += trade.Quantity

		takerWallet.Locked[pair.BaseDenom] -= trade.Quantity
		takerWallet.Balances[pair.BaseDenom] -= trade.Quantity
		takerWallet.Balances[pair.QuoteDenom] += quoteAmount - trade.TakerFee
		makerWallet.Balances[pair.QuoteDenom] -= trade.MakerFee
	}

	k.updateDailyLimit(makerWallet.UserID, quoteAmount)
	k.updateDailyLimit(takerWallet.UserID, quoteAmount)
}

// CancelOrder cancels an open order
func (k *Keeper) CancelOrder(orderID, walletID string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	order, ok := k.Orders[orderID]
	if !ok {
		return types.ErrOrderNotFound
	}
	if order.WalletID != walletID {
		return fmt.Errorf("unauthorized: order belongs to different wallet")
	}
	if order.Status == types.OrderStatusFilled {
		return types.ErrOrderFilled
	}
	if order.Status == types.OrderStatusCancelled {
		return types.ErrOrderCancelled
	}

	wallet := k.Wallets[walletID]
	pair := k.Pairs[order.PairID]

	if order.Side == types.OrderSideBuy {
		lockAmount := (order.Price * order.RemainingQty) / 1_000_000
		wallet.Locked[pair.QuoteDenom] -= lockAmount
	} else {
		wallet.Locked[pair.BaseDenom] -= order.RemainingQty
	}

	order.Status = types.OrderStatusCancelled
	order.UpdatedAt = time.Now().UTC()
	k.removeFromBook(order)
	k.save()
	return nil
}

func (k *Keeper) removeFromBook(order *types.Order) {
	if order.Side == types.OrderSideBuy {
		book := k.BuyBook[order.PairID]
		for i, o := range book {
			if o.ID == order.ID {
				k.BuyBook[order.PairID] = append(book[:i], book[i+1:]...)
				break
			}
		}
	} else {
		book := k.SellBook[order.PairID]
		for i, o := range book {
			if o.ID == order.ID {
				k.SellBook[order.PairID] = append(book[:i], book[i+1:]...)
				break
			}
		}
	}
}

// GetOrderBook returns the current order book snapshot
func (k *Keeper) GetOrderBook(pairID string, depth int) *types.OrderBookSnapshot {
	k.mu.RLock()
	defer k.mu.RUnlock()

	snapshot := &types.OrderBookSnapshot{
		PairID:    pairID,
		Bids:      make([]types.OrderBookLevel, 0),
		Asks:      make([]types.OrderBookLevel, 0),
		Timestamp: time.Now().UTC(),
	}

	bidLevels := make(map[int64]*types.OrderBookLevel)
	for _, order := range k.BuyBook[pairID] {
		if level, ok := bidLevels[order.Price]; ok {
			level.Quantity += order.RemainingQty
			level.Orders++
		} else {
			bidLevels[order.Price] = &types.OrderBookLevel{
				Price: order.Price, Quantity: order.RemainingQty, Orders: 1,
			}
		}
	}
	for _, level := range bidLevels {
		snapshot.Bids = append(snapshot.Bids, *level)
	}
	sort.Slice(snapshot.Bids, func(i, j int) bool {
		return snapshot.Bids[i].Price > snapshot.Bids[j].Price
	})

	askLevels := make(map[int64]*types.OrderBookLevel)
	for _, order := range k.SellBook[pairID] {
		if level, ok := askLevels[order.Price]; ok {
			level.Quantity += order.RemainingQty
			level.Orders++
		} else {
			askLevels[order.Price] = &types.OrderBookLevel{
				Price: order.Price, Quantity: order.RemainingQty, Orders: 1,
			}
		}
	}
	for _, level := range askLevels {
		snapshot.Asks = append(snapshot.Asks, *level)
	}
	sort.Slice(snapshot.Asks, func(i, j int) bool {
		return snapshot.Asks[i].Price < snapshot.Asks[j].Price
	})

	if depth > 0 {
		if len(snapshot.Bids) > depth {
			snapshot.Bids = snapshot.Bids[:depth]
		}
		if len(snapshot.Asks) > depth {
			snapshot.Asks = snapshot.Asks[:depth]
		}
	}

	if len(k.Trades) > 0 {
		snapshot.LastPrice = k.Trades[len(k.Trades)-1].Price
	}

	return snapshot
}

// GetRecentTrades returns recent trades for a pair
func (k *Keeper) GetRecentTrades(pairID string, limit int) []*types.Trade {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var pairTrades []*types.Trade
	for _, t := range k.Trades {
		if t.PairID == pairID {
			pairTrades = append(pairTrades, t)
		}
	}

	if limit > 0 && len(pairTrades) > limit {
		pairTrades = pairTrades[len(pairTrades)-limit:]
	}
	return pairTrades
}

// GetUserOrders returns all orders for a wallet
func (k *Keeper) GetUserOrders(walletID string, activeOnly bool) []*types.Order {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var orders []*types.Order
	for _, o := range k.Orders {
		if o.WalletID == walletID {
			if activeOnly && o.Status != types.OrderStatusOpen && o.Status != types.OrderStatusPartial {
				continue
			}
			orders = append(orders, o)
		}
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})
	return orders
}

// SubmitKYC submits a KYC application
func (k *Keeper) SubmitKYC(app *types.KYCApplication) (*types.KYCApplication, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	for _, existing := range k.KYCApps {
		if existing.UserID == app.UserID && existing.Status == types.KYCStatusPending {
			return nil, types.ErrKYCAlreadyPending
		}
	}

	wallet, ok := k.Wallets[app.WalletID]
	if !ok {
		return nil, types.ErrWalletNotFound
	}
	if wallet.KYCTier >= app.RequestedTier {
		return nil, types.ErrKYCAlreadyApproved
	}

	k.KYCSeq++
	app.ID = fmt.Sprintf("FTG-KYC-%06d", k.KYCSeq)
	app.Status = types.KYCStatusPending
	app.CurrentTier = wallet.KYCTier
	app.SubmittedAt = time.Now().UTC()

	k.runAMLChecks(app)

	// Auto-approve basic tier if clean
	if app.RequestedTier == types.KYCTierBasic && app.AMLRiskScore < 30 && !app.PEPCheck && !app.SanctionsCheck {
		now := time.Now().UTC()
		app.Status = types.KYCStatusApproved
		app.ApprovedAt = &now
		wallet.KYCTier = types.KYCTierBasic
	}

	if app.SanctionsCheck {
		app.Status = types.KYCStatusRejected
		app.RejectedReason = "OFAC/EU sanctions list match"
		wallet.Active = false
	}

	k.KYCApps[app.ID] = app
	k.save()
	return app, nil
}

// ApproveKYC manually approves a KYC application
func (k *Keeper) ApproveKYC(applicationID string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	app, ok := k.KYCApps[applicationID]
	if !ok {
		return fmt.Errorf("KYC application %s not found", applicationID)
	}
	if app.Status != types.KYCStatusPending {
		return fmt.Errorf("application is not pending (status: %s)", app.Status)
	}

	now := time.Now().UTC()
	app.Status = types.KYCStatusApproved
	app.ApprovedAt = &now
	app.ReviewedAt = &now

	wallet := k.Wallets[app.WalletID]
	wallet.KYCTier = app.RequestedTier

	k.save()
	return nil
}

// RejectKYC rejects a KYC application
func (k *Keeper) RejectKYC(applicationID, reason string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	app, ok := k.KYCApps[applicationID]
	if !ok {
		return fmt.Errorf("KYC application %s not found", applicationID)
	}

	now := time.Now().UTC()
	app.Status = types.KYCStatusRejected
	app.RejectedReason = reason
	app.ReviewedAt = &now

	k.save()
	return nil
}

func (k *Keeper) runAMLChecks(app *types.KYCApplication) {
	app.AMLRiskScore = 0

	highRiskCountries := map[string]bool{
		"KP": true, "IR": true, "SY": true, "MM": true,
		"YE": true, "AF": true,
	}
	if highRiskCountries[app.Country] {
		app.AMLRiskScore += 80
		app.SanctionsCheck = true
	}

	app.PEPCheck = false
	if !app.SanctionsCheck {
		app.SanctionsCheck = false
	}
}

func (k *Keeper) checkDailyLimit(wallet *types.ExchangeWallet, price, quantity int64) error {
	if wallet.KYCTier == types.KYCTierInst {
		return nil
	}

	limit := k.Config.TierLimits[wallet.KYCTier]
	if limit == 0 {
		return types.ErrInsufficientKYC
	}

	today := time.Now().UTC().Format("2006-01-02")
	key := wallet.UserID + "/" + today

	dailyLimit, ok := k.DailyLimits[key]
	if !ok {
		dailyLimit = &types.DailyLimit{UserID: wallet.UserID, Date: today}
		k.DailyLimits[key] = dailyLimit
	}

	orderValue := (price * quantity) / 1_000_000
	if dailyLimit.TotalVolume+orderValue > limit {
		return types.ErrDailyLimitExceeded
	}

	return nil
}

func (k *Keeper) updateDailyLimit(userID string, volume int64) {
	today := time.Now().UTC().Format("2006-01-02")
	key := userID + "/" + today

	dailyLimit, ok := k.DailyLimits[key]
	if !ok {
		dailyLimit = &types.DailyLimit{UserID: userID, Date: today}
		k.DailyLimits[key] = dailyLimit
	}
	dailyLimit.TotalVolume += volume
}

// GetExchangeStats returns exchange-wide statistics
func (k *Keeper) GetExchangeStats() map[string]interface{} {
	k.mu.RLock()
	defer k.mu.RUnlock()

	activeWallets := 0
	for _, w := range k.Wallets {
		if w.Active {
			activeWallets++
		}
	}

	openOrders := 0
	for _, o := range k.Orders {
		if o.Status == types.OrderStatusOpen || o.Status == types.OrderStatusPartial {
			openOrders++
		}
	}

	var totalVolume int64
	var last24hVolume int64
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	for _, t := range k.Trades {
		vol := (t.Price * t.Quantity) / 1_000_000
		totalVolume += vol
		if t.ExecutedAt.After(cutoff) {
			last24hVolume += vol
		}
	}

	return map[string]interface{}{
		"total_wallets":     len(k.Wallets),
		"active_wallets":    activeWallets,
		"total_orders":      len(k.Orders),
		"open_orders":       openOrders,
		"total_trades":      len(k.Trades),
		"total_volume_usdc": totalVolume,
		"volume_24h_usdc":   last24hVolume,
		"trading_pairs":     len(k.Pairs),
		"trading_enabled":   k.Config.TradingEnabled,
		"registration_open": k.Config.RegistrationOpen,
	}
}

func (k *Keeper) save() {
	os.MkdirAll(filepath.Dir(k.filePath), 0755)
	data, _ := json.MarshalIndent(k, "", "  ")
	os.WriteFile(k.filePath, data, 0644)
}
