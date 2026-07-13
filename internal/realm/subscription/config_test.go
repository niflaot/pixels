package subscription

import (
	"testing"
	"time"
)

// TestLoadConfigReadsSubscriptionEnvironment verifies configured scheduling values.
func TestLoadConfigReadsSubscriptionEnvironment(t *testing.T) {
	t.Setenv("PIXELS_SUBSCRIPTION_TICK_INTERVAL", "2s")
	t.Setenv("PIXELS_SUBSCRIPTION_PAYDAY_INTERVAL", "48h")
	t.Setenv("PIXELS_SUBSCRIPTION_KICKBACK_PERCENTAGE", "0.25")
	t.Setenv("PIXELS_SUBSCRIPTION_PAYDAY_CURRENCY_TYPE", "5")
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("load subscription config: %v", err)
	}
	if config.TickInterval != 2*time.Second || config.PaydayInterval != 48*time.Hour ||
		config.KickbackPercentage != 0.25 || config.PaydayCurrencyType != 5 {
		t.Fatalf("unexpected config %#v", config)
	}
}

// TestNormalizeRestoresInvalidSubscriptionValues verifies conservative defaults.
func TestNormalizeRestoresInvalidSubscriptionValues(t *testing.T) {
	config := (Config{TickInterval: -1, PaydayInterval: -1, KickbackPercentage: 2}).Normalize()
	if config.TickInterval != time.Minute || config.PaydayInterval != 31*24*time.Hour || config.KickbackPercentage != 0.1 {
		t.Fatalf("unexpected normalized config %#v", config)
	}
}
