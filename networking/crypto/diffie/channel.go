package diffie

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	netconn "github.com/niflaot/pixels/networking/connection"
)

// two is the lower bound for accepted Diffie public values.
var two = big.NewInt(2)

// Channel negotiates legacy Diffie-Hellman and opens and seals RC4 streams.
type Channel struct {
	// mutex protects negotiation state and immutable negotiated values.
	mutex sync.RWMutex
	// config stores compatibility behavior.
	config Config
	// rsa stores server RSA key material.
	rsa *rsaKey
	// state stores the connection security phase.
	state netconn.SecurityState
	// prime stores the per-session Diffie modulus.
	prime *big.Int
	// generator stores the per-session Diffie generator.
	generator *big.Int
	// privateKey stores the per-session private value.
	privateKey *big.Int
	// publicKey stores the per-session public value.
	publicKey *big.Int
	// parameters stores signed values returned during initialization.
	parameters Parameters
	// inbound opens client-to-server packets.
	inbound *rc4Cipher
	// outbound seals server-to-client packets when enabled.
	outbound *rc4Cipher
}

// newChannel creates an unstarted per-session compatibility channel.
func newChannel(config Config, rsa *rsaKey) *Channel {
	return &Channel{config: config, rsa: rsa, state: netconn.SecurityPlain}
}

// State returns the current security phase.
func (channel *Channel) State() netconn.SecurityState {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()

	return channel.state
}

// Begin generates and signs fresh Diffie-Hellman values.
func (channel *Channel) Begin(ctx context.Context) error {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	if channel.state != netconn.SecurityPlain {
		return ErrInvalidState
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	prime, generator, privateKey, err := channel.generateValues()
	if err != nil {
		channel.state = netconn.SecurityFailed
		return err
	}
	publicKey := new(big.Int).Exp(generator, privateKey, prime)
	signedPrime, err := channel.rsa.sign([]byte(prime.Text(10)))
	if err != nil {
		channel.state = netconn.SecurityFailed
		return err
	}
	signedGenerator, err := channel.rsa.sign([]byte(generator.Text(10)))
	if err != nil {
		channel.state = netconn.SecurityFailed
		return err
	}
	channel.prime = prime
	channel.generator = generator
	channel.privateKey = privateKey
	channel.publicKey = publicKey
	channel.parameters = Parameters{EncryptedPrime: signedPrime, EncryptedGenerator: signedGenerator}
	channel.state = netconn.SecurityNegotiating

	return nil
}

// Parameters returns signed initialization values after Begin.
func (channel *Channel) Parameters() (Parameters, error) {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	if channel.state != netconn.SecurityNegotiating {
		return Parameters{}, ErrInvalidState
	}

	return channel.parameters, nil
}

// Complete consumes the protected client public key and prepares RC4 streams.
func (channel *Channel) Complete(ctx context.Context, publicKey PublicKey) (Result, error) {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	if channel.state != netconn.SecurityNegotiating || channel.inbound != nil {
		return Result{}, ErrInvalidState
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	decrypted, err := channel.rsa.decrypt(publicKey.Encrypted)
	if err != nil {
		channel.state = netconn.SecurityFailed
		return Result{}, fmt.Errorf("decrypt client public key: %w", err)
	}
	clientPublic, ok := new(big.Int).SetString(string(decrypted), 10)
	if !ok || !channel.validPublicKey(clientPublic) {
		channel.state = netconn.SecurityFailed
		return Result{}, ErrInvalidPublicKey
	}
	sharedKey := new(big.Int).Exp(clientPublic, channel.privateKey, channel.prime).Bytes()
	if len(sharedKey) == 0 {
		channel.state = netconn.SecurityFailed
		return Result{}, ErrInvalidSharedKey
	}
	inbound, err := newRC4Cipher(sharedKey)
	if err != nil {
		channel.state = netconn.SecurityFailed
		return Result{}, err
	}
	var outbound *rc4Cipher
	if channel.config.ServerClientEncryption {
		outbound, err = newRC4Cipher(sharedKey)
		if err != nil {
			channel.state = netconn.SecurityFailed
			return Result{}, err
		}
	}
	encryptedPublic, err := channel.rsa.sign([]byte(channel.publicKey.Text(10)))
	if err != nil {
		channel.state = netconn.SecurityFailed
		return Result{}, err
	}
	channel.inbound = inbound
	channel.outbound = outbound

	return Result{
		PublicKey:              PublicKey{Encrypted: encryptedPublic},
		ServerClientEncryption: channel.config.ServerClientEncryption,
	}, nil
}

// Activate marks prepared RC4 streams ready after the plaintext completion write.
func (channel *Channel) Activate() error {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	if channel.state != netconn.SecurityNegotiating || channel.inbound == nil {
		return ErrInvalidState
	}
	channel.state = netconn.SecurityReady

	return nil
}

// Open decrypts one client-to-server transport fragment.
func (channel *Channel) Open(source []byte) ([]byte, error) {
	channel.mutex.RLock()
	state, inbound := channel.state, channel.inbound
	channel.mutex.RUnlock()
	if state != netconn.SecurityReady || inbound == nil {
		return nil, ErrInvalidState
	}

	return inbound.apply(source), nil
}

// Seal encrypts one server-to-client transport fragment when requested.
func (channel *Channel) Seal(source []byte) ([]byte, error) {
	channel.mutex.RLock()
	state, outbound := channel.state, channel.outbound
	channel.mutex.RUnlock()
	if state != netconn.SecurityReady {
		return nil, ErrInvalidState
	}
	if outbound == nil {
		return append([]byte(nil), source...), nil
	}

	return outbound.apply(source), nil
}

// Close erases active cipher references and marks the channel failed.
func (channel *Channel) Close(netconn.Reason) error {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	channel.privateKey = nil
	channel.inbound = nil
	channel.outbound = nil
	channel.state = netconn.SecurityFailed

	return nil
}

// generateValues creates compatible prime, generator, and private values.
func (channel *Channel) generateValues() (*big.Int, *big.Int, *big.Int, error) {
	prime, err := rand.Prime(rand.Reader, channel.config.PrimeBits)
	if err != nil {
		return nil, nil, nil, err
	}
	generator, err := rand.Prime(rand.Reader, channel.config.PrimeBits)
	if err != nil {
		return nil, nil, nil, err
	}
	if generator.Cmp(prime) >= 0 {
		prime, generator = generator, prime
	}
	privateKey, err := rand.Prime(rand.Reader, channel.config.PrivateBits)

	return prime, generator, privateKey, err
}

// validPublicKey rejects trivial or out-of-group client values.
func (channel *Channel) validPublicKey(value *big.Int) bool {
	return value != nil && value.Cmp(two) >= 0 && value.Cmp(new(big.Int).Sub(channel.prime, two)) <= 0
}
