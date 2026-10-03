package models

import (
	"encoding/json"
	"testing"
)

func TestRestoredAPIContract(t *testing.T) {
	var event APIEvent
	err := json.Unmarshal([]byte(`{"id":269,"date":"2006-10-04","title":"Launch of Wikileaks","description":"History","url_path":"/2006-10-04/launch-of-wikileaks/","media":"[\"https://example.org/image.webp\"]","references":null,"tags":null}`), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Date.Format("2006-01-02") != "2006-10-04" || event.ID != 269 || len(event.Media) != 1 || len(event.References) != 0 {
		t.Fatalf("incorrect event: %+v", event)
	}
}

func TestInvalidDateRejected(t *testing.T) {
	for _, date := range []string{"", "2006-02-30", "2006-10-04T00:00:00Z"} {
		var event APIEvent
		if json.Unmarshal([]byte(`{"id":1,"date":"`+date+`"}`), &event) == nil {
			t.Fatalf("accepted invalid date %q", date)
		}
	}
}
