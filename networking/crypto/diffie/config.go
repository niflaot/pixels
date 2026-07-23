package diffie

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	netconn "github.com/niflaot/pixels/networking/connection"
)

const (
	// DefaultPrimeBits preserves compatibility with the legacy Habbo handshake.
	DefaultPrimeBits = 128
	// DefaultPrivateBits preserves compatibility with the legacy Habbo handshake.
	DefaultPrivateBits = 128
)

// Config controls the optional legacy Diffie-Hellman and RC4 compatibility layer.
type Config struct {
	// Enabled allows clients to negotiate the legacy in-protocol cipher.
	Enabled bool `env:"PIXELS_DIFFIE_ENABLED" envDefault:"false"`
	// Required rejects authentication when the client did not negotiate Diffie.
	Required bool `env:"PIXELS_DIFFIE_REQUIRED" envDefault:"false"`
	// RSAExponent is the hexadecimal public RSA exponent shared with the client.
	RSAExponent string `env:"PIXELS_DIFFIE_RSA_EXPONENT" envDefault:"3"`
	// RSAModulus is the hexadecimal RSA modulus shared with the client.
	RSAModulus string `env:"PIXELS_DIFFIE_RSA_MODULUS" envDefault:""`
	// RSAPrivateExponent is the hexadecimal RSA private exponent kept by Pixels.
	RSAPrivateExponent string `env:"PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT" envDefault:""`
	// PrimeBits controls the generated Diffie-Hellman prime size.
	PrimeBits int `env:"PIXELS_DIFFIE_PRIME_BITS" envDefault:"128"`
	// PrivateBits controls the generated Diffie-Hellman private value size.
	PrivateBits int `env:"PIXELS_DIFFIE_PRIVATE_BITS" envDefault:"128"`
	// ServerClientEncryption enables RC4 for server-to-client packets.
	ServerClientEncryption bool `env:"PIXELS_DIFFIE_SERVER_CLIENT_ENCRYPTION" envDefault:"true"`
}

// LoadConfig reads legacy Diffie-Hellman compatibility configuration.
func LoadConfig() (Config, error) {
	return env.ParseAs[Config]()
}

// Normalize fills defensive defaults for manually constructed configurations.
func (config Config) Normalize() Config {
	if config.RSAExponent == "" {
		config.RSAExponent = "3"
	}
	if config.PrimeBits <= 0 {
		config.PrimeBits = DefaultPrimeBits
	}
	if config.PrivateBits <= 0 {
		config.PrivateBits = DefaultPrivateBits
	}

	return config
}

// SecurityPolicy returns the explicit authentication security policy.
func (config Config) SecurityPolicy() netconn.SecurityPolicy {
	if config.Required {
		return netconn.SecurityPolicy{Mode: netconn.SecurityRequired}
	}

	return netconn.DefaultSecurityPolicy()
}

// Validate checks configuration combinations without enabling legacy crypto implicitly.
func (config Config) Validate() error {
	config = config.Normalize()
	if config.Required && !config.Enabled {
		return fmt.Errorf("%w: required diffie must be enabled", ErrInvalidConfig)
	}
	if !config.Enabled {
		return nil
	}
	if config.RSAModulus == "" || config.RSAPrivateExponent == "" {
		return fmt.Errorf("%w: RSA modulus and private exponent are required", ErrInvalidConfig)
	}
	if config.PrimeBits < 128 || config.PrimeBits > 512 {
		return fmt.Errorf("%w: prime bits must be between 128 and 512", ErrInvalidConfig)
	}
	if config.PrivateBits < 128 || config.PrivateBits > 512 {
		return fmt.Errorf("%w: private bits must be between 128 and 512", ErrInvalidConfig)
	}

	return nil
}
