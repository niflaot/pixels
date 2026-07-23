package handshake

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"

	"github.com/niflaot/pixels/networking/codec"
	netconn "github.com/niflaot/pixels/networking/connection"
	"github.com/niflaot/pixels/networking/crypto/diffie"
	indiffiecomplete "github.com/niflaot/pixels/networking/inbound/handshake/diffie/complete"
	outdiffiecomplete "github.com/niflaot/pixels/networking/outbound/handshake/diffie/complete"
	outdiffieinit "github.com/niflaot/pixels/networking/outbound/handshake/diffie/init"
)

// TestDiffieHandshakeCompletesAndActivatesRC4 verifies the handler wire flow.
func TestDiffieHandshakeCompletesAndActivatesRC4(t *testing.T) {
	factory, privateKey := testNegotiationFactory(t)
	sent := make([]codec.Packet, 0, 2)
	session := testSessionWithFactory(t, factory, func(packet codec.Packet) {
		sent = append(sent, packet)
	})
	if err := session.Receive(context.Background(), releasePacket(t)); err != nil {
		t.Fatalf("receive release: %v", err)
	}
	if err := session.Receive(context.Background(), diffieInitPacket(t)); err != nil {
		t.Fatalf("receive diffie init: %v", err)
	}
	if len(sent) != 1 || sent[0].Header != outdiffieinit.Header {
		t.Fatalf("expected one init response, got %#v", sent)
	}
	values, err := codec.DecodePacketExact(sent[0], outdiffieinit.Definition)
	if err != nil {
		t.Fatalf("decode init response: %v", err)
	}
	prime := verifySignedDecimal(t, privateKey, values[0].String)
	generator := verifySignedDecimal(t, privateKey, values[1].String)
	clientPrivate := big.NewInt(7919)
	clientPublic := new(big.Int).Exp(generator, clientPrivate, prime)
	protected, err := rsa.EncryptPKCS1v15(rand.Reader, &privateKey.PublicKey, []byte(clientPublic.Text(10)))
	if err != nil {
		t.Fatalf("protect client public key: %v", err)
	}
	packet, err := codec.NewPacket(
		indiffiecomplete.Header,
		indiffiecomplete.Definition,
		codec.String(hex.EncodeToString(protected)),
	)
	if err != nil {
		t.Fatalf("create completion packet: %v", err)
	}
	if err = session.Receive(context.Background(), packet); err != nil {
		t.Fatalf("receive diffie completion: %v", err)
	}
	if len(sent) != 2 || sent[1].Header != outdiffiecomplete.Header {
		t.Fatalf("expected completion response, got %#v", sent)
	}
	if session.State() != netconn.StateHandshaking || session.SecurityState() != netconn.SecurityReady {
		t.Fatalf("unexpected completed states %d and %d", session.State(), session.SecurityState())
	}
	values, err = codec.DecodePacketExact(sent[1], outdiffiecomplete.Definition)
	if err != nil {
		t.Fatalf("decode completion response: %v", err)
	}
	serverPublic := verifySignedDecimal(t, privateKey, values[0].String)
	if !values[1].Boolean || new(big.Int).Exp(serverPublic, clientPrivate, prime).Sign() <= 0 {
		t.Fatal("expected encrypted server stream and a usable shared key")
	}
}

// testNegotiationFactory generates matching server and test-client RSA values.
func testNegotiationFactory(t *testing.T) (*diffie.Factory, *rsa.PrivateKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	factory, err := diffie.NewFactory(diffie.Config{
		Enabled:                true,
		RSAExponent:            strconv.FormatInt(int64(privateKey.E), 16),
		RSAModulus:             privateKey.N.Text(16),
		RSAPrivateExponent:     privateKey.D.Text(16),
		ServerClientEncryption: true,
	})
	if err != nil {
		t.Fatalf("new Diffie factory: %v", err)
	}

	return factory, privateKey
}

// verifySignedDecimal opens a legacy type-one RSA server signature.
func verifySignedDecimal(t *testing.T, privateKey *rsa.PrivateKey, encoded string) *big.Int {
	t.Helper()
	signature, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	block := new(big.Int).Exp(new(big.Int).SetBytes(signature), big.NewInt(int64(privateKey.E)), privateKey.N).Bytes()
	separator := 1
	for separator < len(block) && block[separator] == 0xff {
		separator++
	}
	if len(block) < 11 || block[0] != 1 || separator < 9 || separator >= len(block) || block[separator] != 0 {
		t.Fatalf("invalid signed block %x", block)
	}
	value, ok := new(big.Int).SetString(string(block[separator+1:]), 10)
	if !ok {
		t.Fatalf("invalid signed decimal %q", block[separator+1:])
	}

	return value
}
