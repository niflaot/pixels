package diffie

import (
	"context"
	"errors"
	"math/big"
	"testing"

	netconn "github.com/niflaot/pixels/networking/connection"
)

const (
	// legacyExponent is the public exponent for the compatibility fixture.
	legacyExponent = "3"
	// legacyModulus is the public modulus for the compatibility fixture.
	legacyModulus = "86851dd364d5c5cece3c883171cc6ddc5760779b992482bd1e20dd296888df91b33b936a7b93f06d29e8870f703a216257dec7c81de0058fea4cc5116f75e6efc4e9113513e45357dc3fd43d4efab5963ef178b78bd61e81a14c603b24c8bcce0a12230b320045498edc29282ff0603bc7b7dae8fc1b05b52b2f301a9dc783b7"
	// legacyPrivateExponent is the private exponent for the compatibility fixture.
	legacyPrivateExponent = "59ae13e243392e89ded305764bdd9e92e4eafa67bb6dac7e1415e8c645b0950bccd26246fd0d4af37145af5fa026c0ec3a94853013eaae5ff1888360f4f9449ee023762ec195dff3f30ca0b08b8c947e3859877b5d7dced5c8715c58b53740b84e11fbc71349a27c31745fcefeeea57cff291099205e230e0c7c27e8e1c0512b"
)

// TestChannelNegotiatesAndEncryptsBothDirections verifies the complete wire contract.
func TestChannelNegotiatesAndEncryptsBothDirections(t *testing.T) {
	factory := testFactory(t, true)
	channel, err := factory.New()
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	if err = channel.Begin(context.Background()); err != nil {
		t.Fatalf("begin channel: %v", err)
	}
	parameters, err := channel.Parameters()
	if err != nil {
		t.Fatalf("channel parameters: %v", err)
	}
	prime := verifiedDecimal(t, factory.rsa, parameters.EncryptedPrime)
	generator := verifiedDecimal(t, factory.rsa, parameters.EncryptedGenerator)
	clientPrivate := big.NewInt(7919)
	clientPublic := new(big.Int).Exp(generator, clientPrivate, prime)
	encryptedClientPublic := publicEncrypt(t, factory.rsa, []byte(clientPublic.Text(10)))

	result, err := channel.Complete(context.Background(), PublicKey{Encrypted: encryptedClientPublic})
	if err != nil {
		t.Fatalf("complete channel: %v", err)
	}
	if channel.State() != netconn.SecurityNegotiating {
		t.Fatalf("expected negotiating state, got %d", channel.State())
	}
	serverPublic := verifiedDecimal(t, factory.rsa, result.PublicKey.Encrypted)
	shared := new(big.Int).Exp(serverPublic, clientPrivate, prime).Bytes()
	clientOutbound, err := newRC4Cipher(shared)
	if err != nil {
		t.Fatalf("client outbound cipher: %v", err)
	}
	clientInbound, err := newRC4Cipher(shared)
	if err != nil {
		t.Fatalf("client inbound cipher: %v", err)
	}
	if err = channel.Activate(); err != nil {
		t.Fatalf("activate channel: %v", err)
	}

	plainInbound := []byte("client frame")
	opened, err := channel.Open(clientOutbound.apply(plainInbound))
	if err != nil || string(opened) != string(plainInbound) {
		t.Fatalf("open frame: %q %v", opened, err)
	}
	plainOutbound := []byte("server frame")
	sealed, err := channel.Seal(plainOutbound)
	if err != nil || string(clientInbound.apply(sealed)) != string(plainOutbound) {
		t.Fatalf("seal frame: %x %v", sealed, err)
	}
}

// TestChannelLeavesServerTrafficPlainWhenConfigured verifies one-way compatibility.
func TestChannelLeavesServerTrafficPlainWhenConfigured(t *testing.T) {
	factory := testFactory(t, false)
	channel, result := completeChannel(t, factory)
	if result.ServerClientEncryption {
		t.Fatal("expected server-to-client encryption disabled")
	}
	if err := channel.Activate(); err != nil {
		t.Fatalf("activate channel: %v", err)
	}
	source := []byte("plain response")
	sealed, err := channel.Seal(source)
	if err != nil || string(sealed) != string(source) {
		t.Fatalf("expected plain response, got %q %v", sealed, err)
	}
}

// TestChannelRejectsTrivialClientPublicKey verifies group validation.
func TestChannelRejectsTrivialClientPublicKey(t *testing.T) {
	factory := testFactory(t, true)
	channel, err := factory.New()
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	if err = channel.Begin(context.Background()); err != nil {
		t.Fatalf("begin channel: %v", err)
	}
	encrypted := publicEncrypt(t, factory.rsa, []byte("1"))
	if _, err = channel.Complete(context.Background(), PublicKey{Encrypted: encrypted}); err != ErrInvalidPublicKey {
		t.Fatalf("expected invalid public key, got %v", err)
	}
}

// TestChannelRejectsInvalidStatesAndCloses verifies lifecycle guards.
func TestChannelRejectsInvalidStatesAndCloses(t *testing.T) {
	factory := testFactory(t, true)
	channel, err := factory.New()
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	if _, err = channel.Parameters(); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected parameters state error, got %v", err)
	}
	if err = channel.Activate(); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected activation state error, got %v", err)
	}
	if _, err = channel.Open([]byte("plain")); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected open state error, got %v", err)
	}
	if _, err = channel.Seal([]byte("plain")); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected seal state error, got %v", err)
	}
	if err = channel.Begin(context.Background()); err != nil {
		t.Fatalf("begin channel: %v", err)
	}
	if err = channel.Begin(context.Background()); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected begin state error, got %v", err)
	}
	if err = channel.Close(netconn.Reason{}); err != nil {
		t.Fatalf("close channel: %v", err)
	}
	if channel.State() != netconn.SecurityFailed {
		t.Fatalf("expected failed closed state, got %d", channel.State())
	}
}

// testFactory creates an enabled compatibility factory.
func testFactory(t *testing.T, serverClient bool) *Factory {
	t.Helper()
	factory, err := NewFactory(Config{
		Enabled:                true,
		RSAExponent:            legacyExponent,
		RSAModulus:             legacyModulus,
		RSAPrivateExponent:     legacyPrivateExponent,
		PrimeBits:              128,
		PrivateBits:            128,
		ServerClientEncryption: serverClient,
	})
	if err != nil {
		t.Fatalf("new factory: %v", err)
	}

	return factory
}

// completeChannel completes a channel with a deterministic client private value.
func completeChannel(t *testing.T, factory *Factory) (*Channel, Result) {
	t.Helper()
	channel, err := factory.New()
	if err != nil {
		t.Fatalf("new channel: %v", err)
	}
	if err = channel.Begin(context.Background()); err != nil {
		t.Fatalf("begin channel: %v", err)
	}
	parameters, err := channel.Parameters()
	if err != nil {
		t.Fatalf("channel parameters: %v", err)
	}
	prime := verifiedDecimal(t, factory.rsa, parameters.EncryptedPrime)
	generator := verifiedDecimal(t, factory.rsa, parameters.EncryptedGenerator)
	clientPublic := new(big.Int).Exp(generator, big.NewInt(7919), prime)
	encrypted := publicEncrypt(t, factory.rsa, []byte(clientPublic.Text(10)))
	result, err := channel.Complete(context.Background(), PublicKey{Encrypted: encrypted})
	if err != nil {
		t.Fatalf("complete channel: %v", err)
	}

	return channel, result
}

// verifiedDecimal verifies a signed decimal Diffie value.
func verifiedDecimal(t *testing.T, key *rsaKey, encrypted string) *big.Int {
	t.Helper()
	decoded, err := key.verify(encrypted)
	if err != nil {
		t.Fatalf("verify value: %v", err)
	}
	value, ok := new(big.Int).SetString(string(decoded), 10)
	if !ok {
		t.Fatalf("parse decimal value %q", decoded)
	}

	return value
}

// publicEncrypt performs client-side PKCS#1 type 2 RSA encryption.
func publicEncrypt(t *testing.T, key *rsaKey, source []byte) string {
	t.Helper()
	padded, err := key.pad(source, 2)
	if err != nil {
		t.Fatalf("pad public value: %v", err)
	}

	return bytesHex(key.transform(padded, key.exponent))
}

// bytesHex returns lower-case hexadecimal bytes.
func bytesHex(source []byte) string {
	const alphabet = "0123456789abcdef"
	result := make([]byte, len(source)*2)
	for index, value := range source {
		result[index*2] = alphabet[value>>4]
		result[index*2+1] = alphabet[value&0x0f]
	}

	return string(result)
}
