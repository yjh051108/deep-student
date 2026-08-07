package crypto

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(filepath.Join(dir, "keys"))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello deepstudent-go 🧠")
	ct, err := m.Encrypt("A", plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct, plain) {
		t.Fatal("ciphertext equals plaintext")
	}
	pt, err := m.Decrypt("A", ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, plain) {
		t.Fatalf("mismatch: %s", pt)
	}
}

func TestSwitchSlot(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SwitchSlot("B"); err != nil {
		t.Fatal(err)
	}
	cur, _ := m.ActiveSlot()
	if cur != "B" {
		t.Fatalf("active=%s", cur)
	}
}

func TestDeriveKey(t *testing.T) {
	k1 := DeriveKey("p@ss", []byte("1234567890123456"))
	k2 := DeriveKey("p@ss", []byte("1234567890123456"))
	if !bytes.Equal(k1, k2) {
		t.Fatal("kdf not deterministic")
	}
}
