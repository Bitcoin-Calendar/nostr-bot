package bot

import (
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"strings"
	"testing"
)

func TestKeyFormats(t *testing.T) {
	t.Setenv("BOT_API_KEY", "test")
	key := nostr.GeneratePrivateKey()
	nsec, _ := nip19.EncodePrivateKey(key)
	for _, v := range []string{key, nsec} {
		t.Setenv("NOSTR_PRIVATE_KEY_EN", v)
		c, err := LoadConfig(true)
		if err != nil || c.PrivateKey != key {
			t.Fatalf("key parsing failed: %v", err)
		}
	}
	secret := "invalid-private-secret"
	t.Setenv("NOSTR_PRIVATE_KEY_EN", secret)
	if _, err := LoadConfig(true); err == nil || strings.Contains(err.Error(), secret) {
		t.Fatal("invalid key accepted or leaked")
	}
	for _, invalid := range []string{strings.Repeat("0", 64), strings.Repeat("f", 64)} {
		t.Setenv("NOSTR_PRIVATE_KEY_EN", invalid)
		if _, err := LoadConfig(true); err == nil {
			t.Fatal("invalid scalar accepted")
		}
	}
	if _, err := LoadConfig(false); err != nil {
		t.Fatal("preview requires key")
	}
}
