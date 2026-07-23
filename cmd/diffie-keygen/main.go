// Package main generates an RSA compatibility key triple for Pixels and Nitro.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
)

// output contains generated compatibility configuration values.
type output struct {
	// exponent is the hexadecimal RSA public exponent.
	exponent string
	// modulus is the hexadecimal RSA modulus.
	modulus string
	// privateExponent is the hexadecimal server-only RSA private exponent.
	privateExponent string
}

// main generates and prints one compatibility key triple.
func main() {
	bits := flag.Int("bits", 1024, "RSA key size in bits")
	flag.Parse()
	values, err := generate(*bits)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	write(os.Stdout, values)
}

// generate creates a cryptographically random RSA compatibility key.
func generate(bits int) (output, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return output{}, err
	}
	if err = privateKey.Validate(); err != nil {
		return output{}, err
	}

	return output{
		exponent:        strconv.FormatInt(int64(privateKey.E), 16),
		modulus:         privateKey.N.Text(16),
		privateExponent: privateKey.D.Text(16),
	}, nil
}

// write prints server environment and browser configuration blocks.
func write(destination io.Writer, values output) {
	_, _ = fmt.Fprintf(destination, "PIXELS_DIFFIE_RSA_EXPONENT=%s\n", values.exponent)
	_, _ = fmt.Fprintf(destination, "PIXELS_DIFFIE_RSA_MODULUS=%s\n", values.modulus)
	_, _ = fmt.Fprintf(destination, "PIXELS_DIFFIE_RSA_PRIVATE_EXPONENT=%s\n\n", values.privateExponent)
	_, _ = fmt.Fprintln(destination, `"security.diffie.enabled": true,`)
	_, _ = fmt.Fprintf(destination, "\"security.diffie.rsa.modulus\": %q,\n", values.modulus)
	_, _ = fmt.Fprintf(destination, "\"security.diffie.rsa.exponent\": %q\n", values.exponent)
}
