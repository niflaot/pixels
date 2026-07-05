package packet

import "testing"

// TestNewCopiesBody verifies packets own their body bytes.
func TestNewCopiesBody(t *testing.T) {
	body := []byte{1, 2, 3}
	packet := New(7, body)
	body[0] = 9

	if packet.Header != 7 {
		t.Fatalf("expected header 7, got %d", packet.Header)
	}

	if packet.Body[0] != 1 {
		t.Fatalf("expected copied body, got %v", packet.Body)
	}
}

// TestEmpty verifies body presence checks.
func TestEmpty(t *testing.T) {
	packet := New(7, nil)

	if !packet.Empty() {
		t.Fatal("expected nil body packet to be empty")
	}
}
