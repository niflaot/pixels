package diffie

import (
	"errors"
	"testing"

	netconn "github.com/niflaot/pixels/networking/connection"
)

// TestLoadConfigMapsCompatibilityEnvironment verifies every compatibility setting.
func TestLoadConfigMapsCompatibilityEnvironment(t *testing.T) {
	t.Setenv("PIXELS_DIFFIE_ENABLED", "true")
	t.Setenv("PIXELS_DIFFIE_REQUIRED", "true")
	t.Setenv("PIXELS_DIFFIE_RSA_EXPONENT", legacyExponent)
	t.Setenv("PIXELS_DIFFIE_RSA_MODULUS", legacyModulus)
	t.Setenv("PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT", legacyPrivateExponent)
	t.Setenv("PIXELS_DIFFIE_PRIME_BITS", "192")
	t.Setenv("PIXELS_DIFFIE_PRIVATE_BITS", "224")
	t.Setenv("PIXELS_DIFFIE_SERVER_CLIENT_ENCRYPTION", "false")

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !config.Enabled || !config.Required || config.PrimeBits != 192 || config.PrivateBits != 224 {
		t.Fatalf("unexpected compatibility config: %#v", config)
	}
	if config.RSAExponent != legacyExponent || config.RSAModulus != legacyModulus || config.RSAPrivateExponent != legacyPrivateExponent {
		t.Fatal("expected configured RSA values")
	}
	if config.ServerClientEncryption {
		t.Fatal("expected disabled server-to-client encryption")
	}
	if config.SecurityPolicy().Mode != netconn.SecurityRequired {
		t.Fatal("expected explicitly required security")
	}
}

// TestConfigDefaultsToOptionalDisabledCompatibility verifies production is not implicit.
func TestConfigDefaultsToOptionalDisabledCompatibility(t *testing.T) {
	config := Config{}.Normalize()
	if config.Enabled || config.Required {
		t.Fatalf("expected disabled optional compatibility: %#v", config)
	}
	if config.SecurityPolicy().Mode != netconn.SecurityOptional {
		t.Fatal("expected optional authentication security")
	}
}

// TestConfigRejectsRequiredDisabledCompatibility verifies explicit policy consistency.
func TestConfigRejectsRequiredDisabledCompatibility(t *testing.T) {
	err := (Config{Required: true}).Validate()
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

// TestConfigRejectsInvalidEnabledCompatibility verifies key and size bounds.
func TestConfigRejectsInvalidEnabledCompatibility(t *testing.T) {
	cases := []Config{
		{Enabled: true},
		{Enabled: true, RSAModulus: "aa", RSAPrivateExponent: "bb", PrimeBits: 127},
		{Enabled: true, RSAModulus: "aa", RSAPrivateExponent: "bb", PrimeBits: 513},
		{Enabled: true, RSAModulus: "aa", RSAPrivateExponent: "bb", PrivateBits: 127},
		{Enabled: true, RSAModulus: "aa", RSAPrivateExponent: "bb", PrivateBits: 513},
	}
	for _, config := range cases {
		if err := config.Validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("expected invalid config for %#v, got %v", config, err)
		}
	}
}

// TestFactoryHandlesDisabledAndMalformedKeys verifies startup validation.
func TestFactoryHandlesDisabledAndMalformedKeys(t *testing.T) {
	factory, err := NewFactory(Config{})
	if err != nil || factory.Enabled() {
		t.Fatalf("expected disabled factory, got %#v %v", factory, err)
	}
	if _, err = factory.New(); !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected disabled channel error, got %v", err)
	}
	if _, err = NewFactory(Config{
		Enabled:            true,
		RSAExponent:        "invalid",
		RSAModulus:         legacyModulus,
		RSAPrivateExponent: legacyPrivateExponent,
	}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected malformed key error, got %v", err)
	}
}
