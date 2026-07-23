package diffie

import "sync"

// rc4Cipher stores one direction's stateful legacy RC4 stream.
type rc4Cipher struct {
	// mutex serializes stream state mutation.
	mutex sync.Mutex
	// i stores the first RC4 stream index.
	i byte
	// j stores the second RC4 stream index.
	j byte
	// table stores the RC4 permutation.
	table [256]byte
}

// newRC4Cipher initializes a legacy RC4 stream from a shared key.
func newRC4Cipher(key []byte) (*rc4Cipher, error) {
	if len(key) == 0 {
		return nil, ErrInvalidSharedKey
	}
	cipher := &rc4Cipher{}
	for index := range cipher.table {
		cipher.table[index] = byte(index)
	}
	var j byte
	for index := range cipher.table {
		j += cipher.table[index] + key[index%len(key)]
		cipher.table[index], cipher.table[j] = cipher.table[j], cipher.table[index]
	}

	return cipher, nil
}

// apply copies and transforms bytes while advancing this direction's stream.
func (cipher *rc4Cipher) apply(source []byte) []byte {
	cipher.mutex.Lock()
	defer cipher.mutex.Unlock()

	result := append([]byte(nil), source...)
	for index := range result {
		cipher.i++
		cipher.j += cipher.table[cipher.i]
		cipher.table[cipher.i], cipher.table[cipher.j] = cipher.table[cipher.j], cipher.table[cipher.i]
		key := cipher.table[byte(uint16(cipher.table[cipher.i])+uint16(cipher.table[cipher.j]))]
		result[index] ^= key
	}

	return result
}
