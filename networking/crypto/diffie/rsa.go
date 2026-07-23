package diffie

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// rsaKey implements the raw RSA operations used by the legacy Habbo protocol.
type rsaKey struct {
	// exponent stores the public exponent.
	exponent *big.Int
	// modulus stores the shared modulus.
	modulus *big.Int
	// privateExponent stores the server-only private exponent.
	privateExponent *big.Int
	// blockSize stores the modulus size in bytes.
	blockSize int
}

// newRSAKey parses and validates a hexadecimal RSA key triple.
func newRSAKey(exponentHex string, modulusHex string, privateExponentHex string) (*rsaKey, error) {
	exponent, ok := new(big.Int).SetString(strings.TrimPrefix(exponentHex, "0x"), 16)
	if !ok || exponent.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid RSA exponent", ErrInvalidConfig)
	}
	modulus, ok := new(big.Int).SetString(strings.TrimPrefix(modulusHex, "0x"), 16)
	if !ok || modulus.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid RSA modulus", ErrInvalidConfig)
	}
	privateExponent, ok := new(big.Int).SetString(strings.TrimPrefix(privateExponentHex, "0x"), 16)
	if !ok || privateExponent.Sign() <= 0 {
		return nil, fmt.Errorf("%w: invalid RSA private exponent", ErrInvalidConfig)
	}
	key := &rsaKey{
		exponent:        exponent,
		modulus:         modulus,
		privateExponent: privateExponent,
		blockSize:       (modulus.BitLen() + 7) / 8,
	}
	if key.blockSize < 64 {
		return nil, fmt.Errorf("%w: RSA modulus is too small", ErrInvalidConfig)
	}
	if err := key.validatePair(); err != nil {
		return nil, err
	}

	return key, nil
}

// sign applies PKCS#1 v1.5 block type 1 and the private RSA exponent.
func (key *rsaKey) sign(data []byte) (string, error) {
	padded, err := key.pad(data, 1)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(key.transform(padded, key.privateExponent)), nil
}

// decrypt removes public-key encryption with the private RSA exponent.
func (key *rsaKey) decrypt(encoded string) ([]byte, error) {
	block, err := hex.DecodeString(encoded)
	if err != nil || len(block) != key.blockSize {
		return nil, ErrInvalidPublicKey
	}

	return key.unpad(key.transform(block, key.privateExponent), 2)
}

// verify removes a server signature with the public RSA exponent.
func (key *rsaKey) verify(encoded string) ([]byte, error) {
	block, err := hex.DecodeString(encoded)
	if err != nil || len(block) != key.blockSize {
		return nil, ErrInvalidParameters
	}

	return key.unpad(key.transform(block, key.exponent), 1)
}

// transform performs one fixed-width modular exponentiation block.
func (key *rsaKey) transform(block []byte, exponent *big.Int) []byte {
	value := new(big.Int).SetBytes(block)
	transformed := new(big.Int).Exp(value, exponent, key.modulus).Bytes()
	result := make([]byte, key.blockSize)
	copy(result[key.blockSize-len(transformed):], transformed)

	return result
}

// pad constructs one PKCS#1 v1.5 compatibility block.
func (key *rsaKey) pad(data []byte, blockType byte) ([]byte, error) {
	paddingLength := key.blockSize - len(data) - 3
	if paddingLength < 8 {
		return nil, fmt.Errorf("%w: RSA payload is too large", ErrInvalidParameters)
	}
	result := make([]byte, key.blockSize)
	result[1] = blockType
	if blockType == 1 {
		for index := 2; index < 2+paddingLength; index++ {
			result[index] = 0xff
		}
	} else {
		if err := fillNonZero(rand.Reader, result[2:2+paddingLength]); err != nil {
			return nil, err
		}
	}
	copy(result[3+paddingLength:], data)

	return result, nil
}

// unpad validates and removes one PKCS#1 v1.5 compatibility block.
func (key *rsaKey) unpad(block []byte, blockType byte) ([]byte, error) {
	if len(block) != key.blockSize || block[0] != 0 || block[1] != blockType {
		return nil, ErrInvalidParameters
	}
	separator := 2
	for separator < len(block) && block[separator] != 0 {
		if blockType == 1 && block[separator] != 0xff {
			return nil, ErrInvalidParameters
		}
		separator++
	}
	if separator < 10 || separator >= len(block) {
		return nil, ErrInvalidParameters
	}

	return append([]byte(nil), block[separator+1:]...), nil
}

// validatePair proves that the configured public and private values match.
func (key *rsaKey) validatePair() error {
	signature, err := key.sign([]byte("pixels"))
	if err != nil {
		return err
	}
	verified, err := key.verify(signature)
	if err != nil || string(verified) != "pixels" {
		return fmt.Errorf("%w: RSA public and private values do not match", ErrInvalidConfig)
	}

	return nil
}

// fillNonZero fills PKCS#1 type 2 padding without zero bytes.
func fillNonZero(source io.Reader, target []byte) error {
	for index := range target {
		for target[index] == 0 {
			if _, err := io.ReadFull(source, target[index:index+1]); err != nil {
				return err
			}
		}
	}

	return nil
}
