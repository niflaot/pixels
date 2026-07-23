package config

import (
	"testing"

	"github.com/niflaot/pixels/networking/crypto/diffie"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// TestLoadComposesDiffieConfig verifies legacy compatibility configuration.
func TestLoadComposesDiffieConfig(t *testing.T) {
	t.Setenv("PIXELS_DIFFIE_ENABLED", "true")
	t.Setenv("PIXELS_DIFFIE_REQUIRED", "true")
	t.Setenv("PIXELS_DIFFIE_RSA_EXPONENT", "3")
	t.Setenv("PIXELS_DIFFIE_RSA_MODULUS", "abcd")
	t.Setenv("PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT", "1234")

	config, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !config.Diffie.Enabled || !config.Diffie.Required {
		t.Fatalf("expected enabled required Diffie config, got %#v", config.Diffie)
	}
	if config.Diffie.RSAModulus != "abcd" || config.Diffie.RSAPrivateExponent != "1234" {
		t.Fatalf("unexpected Diffie key config %#v", config.Diffie)
	}
}

// TestModuleProvidesDiffieConfig verifies the focused Fx provider.
func TestModuleProvidesDiffieConfig(t *testing.T) {
	clearEnv(t,
		"PIXELS_DIFFIE_ENABLED",
		"PIXELS_DIFFIE_REQUIRED",
		"PIXELS_DIFFIE_RSA_EXPONENT",
		"PIXELS_DIFFIE_RSA_MODULUS",
		"PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT",
		"PIXELS_DIFFIE_PRIME_BITS",
		"PIXELS_DIFFIE_PRIVATE_BITS",
		"PIXELS_DIFFIE_SERVER_CLIENT_ENCRYPTION",
	)
	var provided diffie.Config
	app := fxtest.New(
		t,
		Module,
		fx.Populate(&provided),
	)

	app.RequireStart()
	app.RequireStop()
	expected := diffie.Config{
		RSAExponent:            "3",
		PrimeBits:              diffie.DefaultPrimeBits,
		PrivateBits:            diffie.DefaultPrivateBits,
		ServerClientEncryption: true,
	}
	if provided != expected {
		t.Fatalf("unexpected provided Diffie config %#v", provided)
	}
}
