package bot

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"math/big"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Config struct {
	APIURL, APIKey, WebURL, PrivateKey, StateDir, StartDate string
	Relays                                                  []string
	Location                                                *time.Location
	Gap                                                     time.Duration
}

func LoadConfig(requireKey bool) (Config, error) {
	c := Config{APIURL: value("BOT_API_ENDPOINT", "http://127.0.0.1:3000/api"), APIKey: os.Getenv("BOT_API_KEY"), WebURL: value("WEB_BASE_URL", "https://bitcoin-calendar.org"), StateDir: value("BOT_STATE_DIR", "/var/lib/bitcal-nostr"), StartDate: value("BOT_START_DATE", "2026-10-03")}
	var err error
	c.Location, err = time.LoadLocation(value("BOT_TIMEZONE", "America/New_York"))
	if err != nil {
		return c, fmt.Errorf("invalid BOT_TIMEZONE")
	}
	if _, err = time.Parse("2006-01-02", c.StartDate); err != nil {
		return c, fmt.Errorf("invalid BOT_START_DATE")
	}
	seconds, err := strconv.Atoi(value("BOT_SLEEP_SECONDS", "1800"))
	if err != nil || seconds < 0 || seconds > 86400 {
		return c, fmt.Errorf("invalid BOT_SLEEP_SECONDS")
	}
	c.Gap = time.Duration(seconds) * time.Second
	for _, address := range []string{c.APIURL, c.WebURL} {
		u, e := url.Parse(address)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return c, fmt.Errorf("invalid API or website URL")
		}
	}
	if c.APIKey == "" {
		return c, fmt.Errorf("BOT_API_KEY is required")
	}
	for _, relay := range strings.Split(value("NOSTR_RELAYS", "wss://relay.primal.net,wss://relay.damus.io,wss://nos.lol,wss://relay.snort.social"), ",") {
		relay = strings.TrimSpace(relay)
		u, e := url.Parse(relay)
		if e != nil || u.Host == "" || (u.Scheme != "wss" && u.Scheme != "ws") || u.User != nil {
			return c, fmt.Errorf("invalid NOSTR_RELAYS")
		}
		c.Relays = append(c.Relays, relay)
	}
	if requireKey {
		key := strings.TrimSpace(os.Getenv("NOSTR_PRIVATE_KEY_EN"))
		if strings.HasPrefix(key, "nsec1") {
			prefix, decoded, e := nip19.Decode(key)
			if e != nil || prefix != "nsec" {
				return c, fmt.Errorf("invalid NOSTR_PRIVATE_KEY_EN")
			}
			var ok bool
			key, ok = decoded.(string)
			if !ok {
				return c, fmt.Errorf("invalid NOSTR_PRIVATE_KEY_EN")
			}
		}
		if len(key) != 64 {
			return c, fmt.Errorf("NOSTR_PRIVATE_KEY_EN must be a hex key or nsec")
		}
		raw, e := hex.DecodeString(key)
		if e != nil {
			return c, fmt.Errorf("invalid NOSTR_PRIVATE_KEY_EN")
		}
		number := new(big.Int).SetBytes(raw)
		if number.Sign() <= 0 || number.Cmp(btcec.S256().N) >= 0 {
			return c, fmt.Errorf("invalid NOSTR_PRIVATE_KEY_EN")
		}
		if _, e := nostr.GetPublicKey(key); e != nil {
			return c, fmt.Errorf("invalid NOSTR_PRIVATE_KEY_EN")
		}
		c.PrivateKey = key
	}
	return c, nil
}
func value(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
