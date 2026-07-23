package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestGenerateAndWrite verifies matching server and renderer values are emitted.
func TestGenerateAndWrite(t *testing.T) {
	values, err := generate(1024)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	var destination bytes.Buffer
	write(&destination, values)
	rendered := destination.String()
	for _, expected := range []string{
		"PIXELS_DIFFIE_RSA_EXPONENT=" + values.exponent,
		"PIXELS_DIFFIE_RSA_MODULUS=" + values.modulus,
		"PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT=" + values.privateExponent,
		`"security.diffie.rsa.modulus": "` + values.modulus + `"`,
		`"security.diffie.rsa.exponent": "` + values.exponent + `"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected generated output to contain %q", expected)
		}
	}
}
