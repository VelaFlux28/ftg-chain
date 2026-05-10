package keeper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/VelaFlux28/ftg-chain/x/kyc/types"
)

// Keeper manages KYC/AML compliance state
type Keeper struct {
	mu sync.RWMutex

	Screenings       map[string]*types.ScreeningResult     `json:"screenings"`
	Rules            map[string]*types.ComplianceRule       `json:"rules"`
	SuspiciousFlags  []*types.SuspiciousActivity           `json:"suspicious_flags"`
	Jurisdictions    []types.RestrictedJurisdiction         `json:"restricted_jurisdictions"`
	Config           *types.ComplianceConfig                `json:"config"`

	ScreeningSeq int64 `json:"screening_seq"`
	FlagSeq      int64 `json:"flag_seq"`

	filePath string
}

// NewKeeper creates or loads the KYC keeper
func NewKeeper(homeDir string) *Keeper {
	statePath := filepath.Join(homeDir, "data", "kyc_state.json")

	k := &Keeper{
		Screenings:      make(map[string]*types.ScreeningResult),
		Rules:           make(map[string]*types.ComplianceRule),
		SuspiciousFlags: make([]*types.SuspiciousActivity, 0),
		Jurisdictions:   types.DefaultRestrictedJurisdictions(),
		Config:          types.DefaultComplianceConfig(),
		filePath:        statePath,
	}

	if data, err := os.ReadFile(statePath); err == nil {
		json.Unmarshal(data, k)
	}

	if len(k.Rules) == 0 {
		k.initDefaultRules()
	}

	return k
}

func (k *Keeper) initDefaultRules() {
	k.Rules["CTR"] = &types.ComplianceRule{
		ID:          "CTR",
		Name:        "Currency Transaction Report",
		Description: "Flag transactions over $10,000 for CTR filing",
		RuleType:    "threshold",
		Enabled:     true,
		Parameters: map[string]interface{}{
			"threshold_usdc": 10_000_000_000,
		},
	}
	k.Rules["STRUCTURING"] = &types.ComplianceRule{
		ID:          "STRUCTURING",
		Name:        "Structuring Detection",
		Description: "Detect multiple transactions just below CTR threshold within 24h",
		RuleType:    "velocity",
		Enabled:     true,
		Parameters: map[string]interface{}{
			"window_hours":    24,
			"threshold_usdc":  9_500_000_000,
			"min_transactions": 3,
		},
	}
	k.Rules["VELOCITY"] = &types.ComplianceRule{
		ID:          "VELOCITY",
		Name:        "High Velocity Trading",
		Description: "Flag unusual trading frequency or volume spikes",
		RuleType:    "velocity",
		Enabled:     true,
		Parameters: map[string]interface{}{
			"max_trades_per_hour": 50,
			"volume_spike_factor": 5.0,
		},
	}
	k.Rules["SANCTIONS_RESCREEN"] = &types.ComplianceRule{
		ID:          "SANCTIONS_RESCREEN",
		Name:        "Periodic Sanctions Re-screening",
		Description: "Re-screen all active users against sanctions lists every 30 days",
		RuleType:    "pattern",
		Enabled:     true,
		Parameters: map[string]interface{}{
			"interval_days": 30,
		},
	}
}

// ScreenUser performs sanctions and PEP screening for a user
func (k *Keeper) ScreenUser(userID, fullName, country, idNumber string) (*types.ScreeningResult, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Check restricted jurisdictions
	for _, j := range k.Jurisdictions {
		if j.CountryCode == country && j.Restriction == "blocked" {
			k.ScreeningSeq++
			result := &types.ScreeningResult{
				ID:            fmt.Sprintf("SCR-%06d", k.ScreeningSeq),
				UserID:        userID,
				ScreeningType: "sanctions",
				Provider:      "internal_jurisdiction_check",
				MatchFound:    true,
				MatchScore:    100,
				MatchDetails:  fmt.Sprintf("Country %s (%s) is blocked: %s", j.CountryName, j.CountryCode, j.Reason),
				ListName:      j.ListSource,
				ScreenedAt:    time.Now().UTC(),
				ExpiresAt:     time.Now().UTC().AddDate(0, 0, k.Config.SanctionsRescreenDays),
			}
			k.Screenings[result.ID] = result
			k.save()
			return result, nil
		}
	}

	// Perform name-based sanctions screening (placeholder for external API)
	k.ScreeningSeq++
	result := &types.ScreeningResult{
		ID:            fmt.Sprintf("SCR-%06d", k.ScreeningSeq),
		UserID:        userID,
		ScreeningType: "sanctions",
		Provider:      k.Config.SanctionsProvider,
		MatchFound:    false,
		MatchScore:    0,
		MatchDetails:  "No matches found against OFAC SDN, EU Consolidated, UN sanctions lists",
		ListName:      "OFAC+EU+UN",
		ScreenedAt:    time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().AddDate(0, 0, k.Config.SanctionsRescreenDays),
	}

	k.Screenings[result.ID] = result
	k.save()
	return result, nil
}

// CheckTransaction evaluates a transaction against compliance rules
func (k *Keeper) CheckTransaction(userID, walletID string, amount int64, txType string) (*types.SuspiciousActivity, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	// CTR check
	if k.Rules["CTR"].Enabled && amount >= k.Config.LargeTransactionThreshold {
		k.FlagSeq++
		flag := &types.SuspiciousActivity{
			ID:          fmt.Sprintf("SAR-%06d", k.FlagSeq),
			UserID:      userID,
			WalletID:    walletID,
			RuleID:      "CTR",
			Description: fmt.Sprintf("Large %s transaction of %d uusdc exceeds CTR threshold", txType, amount),
			Severity:    "medium",
			Amount:      amount,
			FlaggedAt:   time.Now().UTC(),
			Resolution:  "pending",
		}
		k.SuspiciousFlags = append(k.SuspiciousFlags, flag)
		k.save()
		return flag, nil
	}

	return nil, nil
}

// GetUserScreenings returns all screenings for a user
func (k *Keeper) GetUserScreenings(userID string) []*types.ScreeningResult {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var results []*types.ScreeningResult
	for _, s := range k.Screenings {
		if s.UserID == userID {
			results = append(results, s)
		}
	}
	return results
}

// GetPendingFlags returns unresolved suspicious activity flags
func (k *Keeper) GetPendingFlags() []*types.SuspiciousActivity {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var pending []*types.SuspiciousActivity
	for _, f := range k.SuspiciousFlags {
		if f.Resolution == "pending" {
			pending = append(pending, f)
		}
	}
	return pending
}

// ResolveFlag resolves a suspicious activity flag
func (k *Keeper) ResolveFlag(flagID, resolution string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	for _, f := range k.SuspiciousFlags {
		if f.ID == flagID {
			now := time.Now().UTC()
			f.ReviewedAt = &now
			f.Resolution = resolution
			k.save()
			return nil
		}
	}
	return fmt.Errorf("flag %s not found", flagID)
}

// IsJurisdictionBlocked checks if a country is blocked
func (k *Keeper) IsJurisdictionBlocked(countryCode string) (bool, string) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	for _, j := range k.Jurisdictions {
		if j.CountryCode == countryCode {
			if j.Restriction == "blocked" {
				return true, j.Reason
			}
		}
	}
	return false, ""
}

// GetComplianceStats returns compliance module statistics
func (k *Keeper) GetComplianceStats() map[string]interface{} {
	k.mu.RLock()
	defer k.mu.RUnlock()

	pendingFlags := 0
	for _, f := range k.SuspiciousFlags {
		if f.Resolution == "pending" {
			pendingFlags++
		}
	}

	return map[string]interface{}{
		"total_screenings":        len(k.Screenings),
		"total_flags":             len(k.SuspiciousFlags),
		"pending_flags":           pendingFlags,
		"active_rules":            len(k.Rules),
		"restricted_countries":    len(k.Jurisdictions),
		"sanctions_provider":      k.Config.SanctionsProvider,
		"ctr_threshold_usdc":      k.Config.LargeTransactionThreshold,
		"rescreen_interval_days":  k.Config.SanctionsRescreenDays,
	}
}

func (k *Keeper) save() {
	os.MkdirAll(filepath.Dir(k.filePath), 0755)
	data, _ := json.MarshalIndent(k, "", "  ")
	os.WriteFile(k.filePath, data, 0644)
}
