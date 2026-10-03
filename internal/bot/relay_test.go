package bot

import (
	"context"
	"encoding/json"
	"github.com/coder/websocket"
	"github.com/nbd-wtf/go-nostr"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func relayServer(t *testing.T, accept bool, publications *atomic.Int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			var message []json.RawMessage
			if json.Unmarshal(data, &message) != nil || len(message) < 2 {
				return
			}
			var kind string
			json.Unmarshal(message[0], &kind)
			var response any
			switch kind {
			case "EVENT":
				publications.Add(1)
				var note nostr.Event
				if json.Unmarshal(message[1], &note) != nil {
					t.Error("bad event")
				}
				valid, e := note.CheckSignature()
				if note.Kind != 1 || !valid || e != nil {
					t.Error("not a valid kind 1")
				}
				response = []any{"OK", note.ID, accept, "test acknowledgement"}
			case "REQ":
				var id string
				json.Unmarshal(message[1], &id)
				response = []any{"EOSE", id}
			default:
				continue
			}
			raw, _ := json.Marshal(response)
			if conn.Write(r.Context(), websocket.MessageText, raw) != nil {
				return
			}
		}
	}))
}
func TestRealRelayProtocol(t *testing.T) {
	var calls atomic.Int32
	rejected := relayServer(t, false, &calls)
	defer rejected.Close()
	accepted := relayServer(t, true, &calls)
	defer accepted.Close()
	addresses := []string{strings.Replace(rejected.URL, "http:", "ws:", 1), strings.Replace(accepted.URL, "http:", "ws:", 1)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := CheckRelays(ctx, addresses, io.Discard); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("read-only check published an event")
	}
	note := nostr.Event{Kind: 1, CreatedAt: nostr.Now(), Content: "test", Tags: nostr.Tags{}}
	if err := note.Sign(nostr.GeneratePrivateKey()); err != nil {
		t.Fatal(err)
	}
	count, err := PublishToRelays(addresses, io.Discard)(ctx, note)
	if err != nil || count != 1 || calls.Load() != 2 {
		t.Fatalf("count=%d err=%v calls=%d", count, err, calls.Load())
	}
	count, err = PublishToRelays(addresses[:1], io.Discard)(ctx, note)
	if err == nil || count != 0 {
		t.Fatal("total relay rejection reported success")
	}
}
