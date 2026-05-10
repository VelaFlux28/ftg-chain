package types

import "time"

const (
	ModuleName = "kyc"
	StoreKey   = ModuleName
)

// ScreeningResult represents the result of an AML/sanctions screening
type ScreeningResult struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ScreeningType  string    `json:"screening_type"` // sanctions, pep, adverse_media
	Provider       string    `json:"provider"`       // chainalysis, elliptic, comply_advantage
	MatchFound     bool      `json:"match_found"`
	MatchScore     int       `json:"match_score"`    // 0-100 confidence
	MatchDetails   string    `json:"match_details"`
	ListName       string    `json:"list_name"`      // OFAC SDN, EU Consolidated, UN
	ScreenedAt     time.Time `json:"screened_at"`
	ExpiresAt      time.Time `json:"expires_at"`     // Re-screening required after
}

// ComplianceRule defines an automated compliance check
type ComplianceRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RuleType    string `json:"rule_type"` // threshold, velocity, pattern
	Enabled     bool   `json:"enabled"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// SuspiciousActivity represents a flagged transaction
type SuspiciousActivity struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	WalletID    string    `json:"wallet_id"`
	RuleID      string    `json:"rule_id"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	Amount      int64     `json:"amount"`
	FlaggedAt   time.Time `json:"flagged_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	Resolution  string    `json:"resolution"` // cleared, escalated, frozen
}

// RestrictedJurisdiction defines countries/regions with trading restrictions
type RestrictedJurisdiction struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Restriction string `json:"restriction"` // blocked, enhanced_dd, limited
	Reason      string `json:"reason"`
	ListSource  string `json:"list_source"` // OFAC, EU, FATF
}

// DefaultRestrictedJurisdictions returns the OFAC/FATF blocked list
func DefaultRestrictedJurisdictions() []RestrictedJurisdiction {
	return []RestrictedJurisdiction{
		{CountryCode: "KP", CountryName: "North Korea", Restriction: "blocked", Reason: "OFAC comprehensive sanctions", ListSource: "OFAC"},
		{CountryCode: "IR", CountryName: "Iran", Restriction: "blocked", Reason: "OFAC comprehensive sanctions", ListSource: "OFAC"},
		{CountryCode: "SY", CountryName: "Syria", Restriction: "blocked", Reason: "OFAC comprehensive sanctions", ListSource: "OFAC"},
		{CountryCode: "CU", CountryName: "Cuba", Restriction: "blocked", Reason: "OFAC comprehensive sanctions", ListSource: "OFAC"},
		{CountryCode: "RU", CountryName: "Russia", Restriction: "blocked", Reason: "OFAC sectoral sanctions", ListSource: "OFAC"},
		{CountryCode: "BY", CountryName: "Belarus", Restriction: "blocked", Reason: "EU sanctions", ListSource: "EU"},
		{CountryCode: "MM", CountryName: "Myanmar", Restriction: "enhanced_dd", Reason: "FATF grey list", ListSource: "FATF"},
		{CountryCode: "YE", CountryName: "Yemen", Restriction: "enhanced_dd", Reason: "FATF grey list", ListSource: "FATF"},
		{CountryCode: "AF", CountryName: "Afghanistan", Restriction: "blocked", Reason: "FATF black list", ListSource: "FATF"},
		{CountryCode: "VE", CountryName: "Venezuela", Restriction: "enhanced_dd", Reason: "OFAC targeted sanctions", ListSource: "OFAC"},
	}
}

// ComplianceConfig holds KYC/AML module configuration
type ComplianceConfig struct {
	// Screening providers
	SanctionsProvider string `json:"sanctions_provider"`
	PEPProvider       string `json:"pep_provider"`

	// Thresholds
	LargeTransactionThreshold int64 `json:"large_transaction_threshold"` // USDC amount triggering CTR
	StructuringWindow         int   `json:"structuring_window_hours"`    // Hours to detect structuring
	StructuringThreshold      int64 `json:"structuring_threshold"`       // Total in window

	// Re-screening intervals
	SanctionsRescreenDays int `json:"sanctions_rescreen_days"`
	PEPRescreenDays       int `json:"pep_rescreen_days"`

	// Auto-freeze rules
	AutoFreezeOnSanctions bool `json:"auto_freeze_on_sanctions"`
	AutoFreezeOnHighRisk  bool `json:"auto_freeze_on_high_risk"`
	HighRiskScoreThreshold int  `json:"high_risk_score_threshold"`
}

// DefaultComplianceConfig returns sensible defaults
func DefaultComplianceConfig() *ComplianceConfig {
	return &ComplianceConfig{
		SanctionsProvider:         "internal",
		PEPProvider:               "internal",
		LargeTransactionThreshold: 10_000_000_000, // $10,000 USDC (CTR threshold)
		StructuringWindow:         24,
		StructuringThreshold:      9_500_000_000,  // Just under CTR threshold
		SanctionsRescreenDays:     30,
		PEPRescreenDays:           90,
		AutoFreezeOnSanctions:     true,
		AutoFreezeOnHighRisk:      true,
		HighRiskScoreThreshold:    70,
	}
}
