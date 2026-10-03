package bot

import (
	"calendar-bot/internal/models"
	"context"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"io"
	"time"
)

type Fetch func(string, string, string) ([]models.APIEvent, error)
type Publish func(context.Context, nostr.Event) (int, error)

func Run(ctx context.Context, c Config, now time.Time, fetch Fetch, publish Publish, out io.Writer) error {
	today := now.In(c.Location).Format("2006-01-02")
	if today < c.StartDate {
		fmt.Fprintf(out, "Not before %s; no posts sent\n", c.StartDate)
		return nil
	}
	pubkey, err := nostr.GetPublicKey(c.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid private key")
	}
	store, err := OpenStore(c.StateDir)
	if err != nil {
		return err
	}
	defer store.Close()
	events, err := fetch(now.In(c.Location).Format("01"), now.In(c.Location).Format("02"), "en")
	if err != nil {
		return err
	}
	// Validate the entire day's input before the first irreversible network write.
	notes := make([]nostr.Event, len(events))
	seen := map[uint]bool{}
	for i, event := range events {
		if event.Date.Format("01-02") != now.In(c.Location).Format("01-02") || seen[event.ID] {
			return fmt.Errorf("API returned wrong date or duplicate event %d", event.ID)
		}
		seen[event.ID] = true
		notes[i], err = Format(c, event)
		if err != nil {
			return err
		}
	}
	attempted := false
	for i, event := range events {
		if err = ctx.Err(); err != nil {
			return err
		}
		record, e := store.Load(pubkey, today, event.ID)
		if e != nil {
			return e
		}
		if record != nil && record.Delivered {
			fmt.Fprintf(out, "Already delivered event=%d nostr=%s\n", event.ID, record.Event.ID)
			continue
		}
		if attempted && c.Gap > 0 {
			timer := time.NewTimer(c.Gap)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		if record == nil {
			note := notes[i]
			note.CreatedAt = nostr.Timestamp(now.Unix())
			if err = note.Sign(c.PrivateKey); err != nil {
				return fmt.Errorf("signing event %d failed", event.ID)
			}
			record = &Record{Event: note}
			// Persist before publishing: even an ambiguous network response reuses this ID.
			if err = store.Save(pubkey, today, event.ID, record); err != nil {
				return err
			}
		}
		count, e := publish(ctx, record.Event)
		attempted = true
		if e != nil {
			return e
		}
		if count == 0 {
			return fmt.Errorf("event %d was rejected by every relay", event.ID)
		}
		record.Delivered = true
		if err = store.Save(pubkey, today, event.ID, record); err != nil {
			return err
		}
		fmt.Fprintf(out, "Delivered event=%d nostr=%s relays=%d\n", event.ID, record.Event.ID, count)
	}
	fmt.Fprintf(out, "Completed date=%s events=%d\n", today, len(events))
	return nil
}
