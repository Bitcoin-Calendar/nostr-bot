package bot

import (
	"calendar-bot/internal/models"
	"fmt"
	"github.com/nbd-wtf/go-nostr"
	"net/url"
	"path"
	"strings"
)

func Format(c Config, event models.APIEvent) (nostr.Event, error) {
	if event.ID == 0 || event.Title == "" || event.Description == "" {
		return nostr.Event{}, fmt.Errorf("event %d lacks required text", event.ID)
	}
	// Keep links on the configured site, never accept a host or traversal from data.
	if !strings.HasPrefix(event.URLPath, "/") || strings.HasPrefix(event.URLPath, "//") || strings.ContainsAny(event.URLPath, "?#\\") {
		return nostr.Event{}, fmt.Errorf("event %d has invalid url_path", event.ID)
	}
	parts := strings.Split(strings.Trim(event.URLPath, "/"), "/")
	if len(parts) != 2 || parts[0] != event.Date.Format("2006-01-02") || parts[1] == "" || parts[1] == "." || parts[1] == ".." {
		return nostr.Event{}, fmt.Errorf("event %d has invalid url_path", event.ID)
	}
	link := strings.TrimRight(c.WebURL, "/") + "/en/events/" + parts[0] + "/" + parts[1]
	text := event.Title + "\n\n" + event.Description + "\n\n" + link
	// Only the first explicit event image; no generic OG fallback and no kind 20.
	for _, media := range event.Media {
		u, err := url.Parse(media)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			continue
		}
		switch strings.ToLower(path.Ext(u.Path)) {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".apng":
			text += "\n\n" + media
			return nostr.Event{Kind: 1, Content: text, Tags: nostr.Tags{{"t", "bitcoin"}, {"t", "history"}, {"r", link}}}, nil
		}
	}
	return nostr.Event{Kind: 1, Content: text, Tags: nostr.Tags{{"t", "bitcoin"}, {"t", "history"}, {"r", link}}}, nil
}
