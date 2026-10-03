package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// APIEvent matches the data structure from the API.
type APIEvent struct {
	ID          uint      `json:"ID"`
	Date        time.Time `json:"Date"`
	Title       string    `json:"Title"`
	Description string    `json:"Description"`
	Tags        string    `json:"Tags"`
	Media       []string  `json:"Media"`
	References  []string  `json:"References"`
	Hashtags    []string  `json:"hashtags"`
	Olas        bool      `json:"olas"`
	URLPath     string    `json:"url_path"`
}

// apiEventRaw is an intermediate struct for unmarshalling.
type apiEventRaw struct {
	ID          uint   `json:"ID"`
	Date        string `json:"Date"`
	Title       string `json:"Title"`
	Description string `json:"Description"`
	Tags        string `json:"Tags"`
	Media       string `json:"Media"`
	References  string `json:"References"`
	Olas        bool   `json:"olas"`
	URLPath     string `json:"url_path"`
}

// UnmarshalJSON provides custom unmarshalling logic for APIEvent.
// It handles 'Media' and 'References' fields that can be either a JSON array string or a plain string.
func (ae *APIEvent) UnmarshalJSON(data []byte) error {
	var raw apiEventRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", raw.Date)
	if err != nil {
		return fmt.Errorf("invalid event date: %w", err)
	}
	if raw.ID == 0 {
		return fmt.Errorf("event id must be positive")
	}
	*ae = APIEvent{}
	ae.ID = raw.ID
	ae.Date = date
	ae.URLPath = raw.URLPath
	ae.Title = raw.Title
	ae.Description = raw.Description
	ae.Tags = raw.Tags
	ae.Olas = raw.Olas

	// Unmarshal Media string into []string
	if raw.Media != "" && raw.Media != "[]" {
		if err := json.Unmarshal([]byte(raw.Media), &ae.Media); err != nil {
			ae.Media = []string{raw.Media}
		}
	} else {
		ae.Media = []string{}
	}

	// Unmarshal References string into []string
	if raw.References != "" && raw.References != "[]" {
		if err := json.Unmarshal([]byte(raw.References), &ae.References); err != nil {
			ae.References = []string{raw.References}
		}
	} else {
		ae.References = []string{}
	}

	return nil
}

// APIResponseWrapper represents the full structure of the API response.
type APIResponseWrapper struct {
	Events     []APIEvent  `json:"events"`
	Pagination interface{} `json:"pagination"`
}

// Constants for event types, if needed elsewhere
