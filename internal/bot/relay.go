package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"github.com/nbd-wtf/go-nostr"
	"io"
	"time"
)

// Each connection performs one bounded NIP-01 exchange. Signing remains in
// go-nostr; transport uses the existing websocket dependency directly to avoid
// a shutdown race in the legacy go-nostr relay client.
func exchange(ctx context.Context, address string, request any, eventID string) error {
	conn, _, err := websocket.Dial(ctx, address, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(1 << 20)
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		return err
	}
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		var message []json.RawMessage
		if json.Unmarshal(data, &message) != nil || len(message) < 2 {
			return fmt.Errorf("invalid relay response")
		}
		var kind string
		if json.Unmarshal(message[0], &kind) != nil {
			return fmt.Errorf("invalid relay response type")
		}
		if eventID != "" && kind == "OK" && len(message) >= 3 {
			var id string
			var accepted bool
			if json.Unmarshal(message[1], &id) != nil || json.Unmarshal(message[2], &accepted) != nil {
				return fmt.Errorf("invalid relay acknowledgement")
			}
			if id != eventID {
				continue
			}
			if !accepted {
				return fmt.Errorf("relay rejected event %s", id)
			}
			return nil
		}
		if eventID == "" && (kind == "EVENT" || kind == "EOSE") {
			var subscription string
			if json.Unmarshal(message[1], &subscription) == nil && subscription == "bitcal-probe" {
				return nil
			}
		}
		if kind == "AUTH" || kind == "CLOSED" {
			return fmt.Errorf("relay requires authentication or rejected subscription")
		}
	}
}

func PublishToRelays(relays []string, out io.Writer) Publish {
	return func(ctx context.Context, event nostr.Event) (int, error) {
		successes := 0
		for _, address := range relays {
			if err := ctx.Err(); err != nil {
				return successes, err
			}
			timeout, cancel := context.WithTimeout(ctx, 20*time.Second)
			err := exchange(timeout, address, []any{"EVENT", event}, event.ID)
			cancel()
			if err != nil {
				fmt.Fprintf(out, "Relay failed %s: %v\n", address, err)
			} else {
				successes++
				fmt.Fprintf(out, "Relay accepted %s\n", address)
			}
		}
		if successes == 0 {
			return 0, fmt.Errorf("no relay accepted the event")
		}
		return successes, nil
	}
}

func CheckRelays(ctx context.Context, relays []string, out io.Writer) error {
	failures := 0
	for _, address := range relays {
		timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := exchange(timeout, address, []any{"REQ", "bitcal-probe", map[string]any{"kinds": []int{1}, "limit": 1}}, "")
		cancel()
		if err != nil {
			failures++
			fmt.Fprintf(out, "Relay check failed %s: %v\n", address, err)
		} else {
			fmt.Fprintf(out, "Relay read check passed %s\n", address)
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d relay checks failed (no posts sent)", failures)
	}
	return nil
}
