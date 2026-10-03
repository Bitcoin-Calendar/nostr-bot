package bot

import (
	"bytes"
	"calendar-bot/internal/models"
	"context"
	"errors"
	"github.com/nbd-wtf/go-nostr"
	"io"
	"os"
	"testing"
	"time"
)

func fixture(t *testing.T) (Config, time.Time, models.APIEvent) {
	t.Helper()
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	historical, _ := time.Parse("2006-01-02", "2006-10-04")
	return Config{PrivateKey: nostr.GeneratePrivateKey(), StateDir: t.TempDir(), Location: location, StartDate: "2026-10-04", WebURL: "https://bitcoin-calendar.org"}, time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC), models.APIEvent{ID: 269, Date: historical, Title: "Launch of Wikileaks", Description: "History", URLPath: "/2006-10-04/launch-of-wikileaks/", Media: []string{"https://example.org/photo.webp"}}
}
func TestFormatting(t *testing.T) {
	c, _, event := fixture(t)
	for _, media := range [][]string{{"https://example.org/photo.webp", "https://example.org/second.png"}, nil, {"https://example.org/video.mp4"}} {
		event.Media = media
		note, err := Format(c, event)
		if err != nil {
			t.Fatal(err)
		}
		expected := "Launch of Wikileaks\n\nHistory\n\nhttps://bitcoin-calendar.org/en/events/2006-10-04/launch-of-wikileaks"
		if len(media) == 2 {
			expected += "\n\nhttps://example.org/photo.webp"
		}
		if note.Kind != 1 || note.Content != expected {
			t.Fatalf("unexpected note %+v", note)
		}
	}
	event.URLPath = "//evil.example/path"
	if _, err := Format(c, event); err == nil {
		t.Fatal("accepted invalid permalink")
	}
}
func TestRetryReusesSignedEventAndSkipsDelivered(t *testing.T) {
	c, now, event := fixture(t)
	fetch := func(month, day, lang string) ([]models.APIEvent, error) {
		if month != "10" || day != "04" || lang != "en" {
			t.Fatal("wrong date/language")
		}
		return []models.APIEvent{event}, nil
	}
	var first nostr.Event
	calls := 0
	publish := func(ctx context.Context, note nostr.Event) (int, error) {
		calls++
		ok, err := note.CheckSignature()
		if !ok || err != nil || note.Kind != 1 {
			t.Fatal("invalid signed kind 1")
		}
		// The exact signed note is durable before any network side effect.
		files, err := os.ReadDir(c.StateDir)
		if err != nil || len(files) != 2 {
			t.Fatal("pending note not stored")
		}
		if calls == 1 {
			first = note
			return 0, errors.New("lost acknowledgement")
		}
		if note.ID != first.ID || note.Sig != first.Sig || note.CreatedAt != first.CreatedAt {
			t.Fatal("retry changed event")
		}
		return 4, nil
	}
	if Run(context.Background(), c, now, fetch, publish, io.Discard) == nil {
		t.Fatal("publish failure swallowed")
	}
	if err := Run(context.Background(), c, now.Add(time.Hour), fetch, publish, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), c, now.Add(2*time.Hour), fetch, publish, io.Discard); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("delivered event was republished: calls=%d", calls)
	}
}
func TestPublicationBoundary(t *testing.T) {
	c, now, event := fixture(t)
	cases := []struct {
		name      string
		when      time.Time
		events    []models.APIEvent
		wantError bool
	}{
		{"before first date", now.Add(-24 * time.Hour), []models.APIEvent{event}, false},
		{"empty day", now, nil, false},
		{"duplicate ID", now, []models.APIEvent{event, event}, true},
	}
	wrong := event
	wrong.Date = wrong.Date.AddDate(0, 0, 1)
	cases = append(cases, struct {
		name      string
		when      time.Time
		events    []models.APIEvent
		wantError bool
	}{"wrong day", now, []models.APIEvent{wrong}, true})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c.StateDir = t.TempDir()
			calls := 0
			err := Run(context.Background(), c, tc.when, func(string, string, string) ([]models.APIEvent, error) { return tc.events, nil }, func(context.Context, nostr.Event) (int, error) { calls++; return 1, nil }, io.Discard)
			if (err != nil) != tc.wantError || calls != 0 {
				t.Fatalf("error=%v publish calls=%d", err, calls)
			}
		})
	}
}
func TestStateLockAndCorruption(t *testing.T) {
	c, now, event := fixture(t)
	store, err := OpenStore(c.StateDir)
	if err != nil {
		t.Fatal(err)
	}
	if another, err := OpenStore(c.StateDir); err == nil {
		another.Close()
		t.Fatal("concurrent lock acquired")
	}
	store.Close()
	fetch := func(string, string, string) ([]models.APIEvent, error) { return []models.APIEvent{event}, nil }
	if err := Run(context.Background(), c, now, fetch, func(context.Context, nostr.Event) (int, error) { return 1, nil }, io.Discard); err != nil {
		t.Fatal(err)
	}
	pubkey, _ := nostr.GetPublicKey(c.PrivateKey)
	store, err = OpenStore(c.StateDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	filename := store.file(pubkey, "2026-10-04", event.ID)
	data, _ := os.ReadFile(filename)
	data = bytes.Replace(data, []byte("History"), []byte("Tampered"), 1)
	os.WriteFile(filename, data, 0600)
	if _, err := store.Load(pubkey, "2026-10-04", event.ID); err == nil {
		t.Fatal("tampered state accepted")
	}
}
func TestNewYorkDate(t *testing.T) {
	c, _, event := fixture(t)
	// UTC is already Oct 5 while New York is still Oct 4.
	now := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	if err := Run(context.Background(), c, now, func(m, d, l string) ([]models.APIEvent, error) {
		if d != "04" {
			t.Fatal("UTC day selected")
		}
		return []models.APIEvent{event}, nil
	}, func(context.Context, nostr.Event) (int, error) { return 1, nil }, io.Discard); err != nil {
		t.Fatal(err)
	}
}
