package crypto

import (
	"strings"
	"testing"
)

func TestArgon2_HashECompare(t *testing.T) {
	h := NewArgon2Hasher()

	hash, err := h.Hash("Senha@123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("formato inesperado: %s", hash)
	}
	if !h.Compare(hash, "Senha@123") {
		t.Fatal("senha correta foi rejeitada")
	}
	if h.Compare(hash, "senha@123") {
		t.Fatal("senha errada foi aceita")
	}
}

func TestArgon2_SaltAleatorio(t *testing.T) {
	h := NewArgon2Hasher()
	a, _ := h.Hash("mesma")
	b, _ := h.Hash("mesma")
	if a == b {
		t.Fatal("dois hashes da mesma senha não podem ser iguais")
	}
}

func TestArgon2_HashMalformado(t *testing.T) {
	h := NewArgon2Hasher()
	for _, enc := range []string{
		"",
		"texto-qualquer",
		"$argon2id$v=19$m=65536,t=1,p=4$salt",
		"$argon2id$v=19$lixo$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=1,p=4$!!!$aGFzaA",
		"$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$",
	} {
		if h.Compare(enc, "x") {
			t.Errorf("hash malformado aceito: %q", enc)
		}
	}
}
