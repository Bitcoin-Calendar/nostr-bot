package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRestoredAPIOverHTTP(t *testing.T) {
	bodies := []struct {
		body      string
		wantError bool
		count     int
	}{
		{`{"events":[{"id":269,"date":"2006-10-04","title":"Title","description":"Text","url_path":"/2006-10-04/title/","media":null,"references":null,"tags":null}]}`, false, 1},
		{`{"events":[]}`, false, 0},
		{`{"events":null}`, true, 0},
		{`{}`, true, 0},
		{`{"events":[{"id":269,"date":"broken"}]}`, true, 0},
	}
	for _, tc := range bodies {
		t.Run(tc.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/events" || r.URL.Query().Get("lang") != "en" || r.URL.Query().Get("day") != "04" || r.Header.Get("X-API-Key") != "test-key" {
					t.Error("incorrect API request")
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client := NewClient(server.URL+"/api", "test-key")
			events, err := client.FetchEvents("10", "04", "en")
			if (err != nil) != tc.wantError || len(events) != tc.count {
				t.Fatalf("events=%v err=%v", events, err)
			}
		})
	}
}
