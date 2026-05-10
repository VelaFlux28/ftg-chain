package keeper

import (
	"os"
	"path/filepath"
	"testing"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "data"), 0755)
	return dir
}

func TestScreenUserCleanCountry(t *testing.T) {
	k := NewKeeper(tempDir(t))

	result, err := k.ScreenUser("user1", "John Smith", "US", "123456789")
	if err != nil {
		t.Fatalf("ScreenUser failed: %v", err)
	}
	if result.MatchFound {
		t.Fatal("US user should not match sanctions")
	}
	if result.MatchScore != 0 {
		t.Fatalf("expected score 0, got %d", result.MatchScore)
	}
}

func TestScreenUserBlockedCountry(t *testing.T) {
	k := NewKeeper(tempDir(t))

	result, err := k.ScreenUser("user2", "Test User", "KP", "000000")
	if err != nil {
		t.Fatalf("ScreenUser failed: %v", err)
	}
	if !result.MatchFound {
		t.Fatal("KP user should match sanctions")
	}
	if result.MatchScore != 100 {
		t.Fatalf("expected score 100, got %d", result.MatchScore)
	}
}

func TestCheckTransactionCTR(t *testing.T) {
	k := NewKeeper(tempDir(t))

	// Below threshold — no flag
	flag, err := k.CheckTransaction("user1", "w1", 5_000_000_000, "buy")
	if err != nil {
		t.Fatalf("CheckTransaction failed: %v", err)
	}
	if flag != nil {
		t.Fatal("should not flag transaction below threshold")
	}

	// Above threshold — should flag
	flag, err = k.CheckTransaction("user1", "w1", 15_000_000_000, "buy")
	if err != nil {
		t.Fatalf("CheckTransaction failed: %v", err)
	}
	if flag == nil {
		t.Fatal("should flag transaction above CTR threshold")
	}
	if flag.Severity != "medium" {
		t.Fatalf("expected medium severity, got %s", flag.Severity)
	}
}

func TestResolveFlag(t *testing.T) {
	k := NewKeeper(tempDir(t))

	// Create a flag
	k.CheckTransaction("user1", "w1", 15_000_000_000, "buy")

	flags := k.GetPendingFlags()
	if len(flags) != 1 {
		t.Fatalf("expected 1 pending flag, got %d", len(flags))
	}

	// Resolve it
	err := k.ResolveFlag(flags[0].ID, "cleared")
	if err != nil {
		t.Fatalf("ResolveFlag failed: %v", err)
	}

	// Should be no more pending
	flags = k.GetPendingFlags()
	if len(flags) != 0 {
		t.Fatalf("expected 0 pending flags after resolution, got %d", len(flags))
	}
}

func TestIsJurisdictionBlocked(t *testing.T) {
	k := NewKeeper(tempDir(t))

	blocked, reason := k.IsJurisdictionBlocked("IR")
	if !blocked {
		t.Fatal("Iran should be blocked")
	}
	if reason == "" {
		t.Fatal("should have a reason")
	}

	blocked, _ = k.IsJurisdictionBlocked("NL")
	if blocked {
		t.Fatal("Netherlands should not be blocked")
	}
}

func TestComplianceStats(t *testing.T) {
	k := NewKeeper(tempDir(t))

	stats := k.GetComplianceStats()
	if stats["active_rules"].(int) != 4 {
		t.Fatalf("expected 4 rules, got %v", stats["active_rules"])
	}
	if stats["restricted_countries"].(int) < 5 {
		t.Fatalf("expected at least 5 restricted countries, got %v", stats["restricted_countries"])
	}
}
