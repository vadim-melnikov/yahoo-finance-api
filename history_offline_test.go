package yahoofinanceapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

// newOfflineServer starts an httptest.Server that answers every request with body
// and points BASE_URL at it. The shared client gets a preset crumb, so it does not
// go to fc.yahoo.com for cookies. Everything is restored when the test ends.
// The returned function reports the query of the last request the server received.
func newOfflineServer(t *testing.T, body []byte) func() url.Values {
	t.Helper()

	var lastQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))

	client := getClient()
	oldBaseURL, oldCrumb := BASE_URL, client.crumb
	BASE_URL = server.URL
	client.crumb = "test-crumb"
	t.Cleanup(func() {
		server.Close()
		BASE_URL = oldBaseURL
		client.crumb = oldCrumb
	})

	return func() url.Values { return lastQuery }
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("Failed to read fixture %s: %v", name, err)
	}
	return data
}

func TestHistoryParamsWithoutEvents(t *testing.T) {
	q := HistoryQuery{Range: "1mo", Interval: "1d", Start: "1596240000", End: "1601424000"}
	params := historyParams(&q)

	if params.Has("events") {
		t.Errorf("Expected no events parameter, got %q", params.Get("events"))
	}
	want := url.Values{
		"range":    {"1mo"},
		"interval": {"1d"},
		"period1":  {"1596240000"},
		"period2":  {"1601424000"},
	}
	if params.Encode() != want.Encode() {
		t.Errorf("Expected params %q, got %q", want.Encode(), params.Encode())
	}
}

func TestHistoryParamsWithEvents(t *testing.T) {
	for _, events := range []string{"split", "div,split"} {
		q := HistoryQuery{Interval: "1d", Start: "1596240000", End: "1601424000", Events: events}
		params := historyParams(&q)
		if got := params.Get("events"); got != events {
			t.Errorf("Expected events parameter %q, got %q", events, got)
		}
	}
}

func TestGetHistoryEventsParameterOffline(t *testing.T) {
	lastQuery := newOfflineServer(t, readFixture(t, "chart_no_events.json"))

	history := newHistory()
	history.SetQuery(HistoryQuery{Start: "2020-08-01", End: "2020-09-30"})
	if _, err := history.GetHistory("TEST"); err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	if lastQuery().Has("events") {
		t.Errorf("Expected no events parameter without Events, got %q", lastQuery().Get("events"))
	}

	history.SetQuery(HistoryQuery{Start: "2020-08-01", End: "2020-09-30", Events: "split"})
	if _, err := history.GetHistory("TEST"); err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	if got := lastQuery().Get("events"); got != "split" {
		t.Errorf("Expected events parameter %q, got %q", "split", got)
	}
}

func TestGetHistoryMalformedJSONOffline(t *testing.T) {
	newOfflineServer(t, []byte(`{"chart": {"result": [`))

	history := newHistory()
	_, err := history.GetHistory("TEST")
	if err == nil {
		t.Fatal("Expected error for malformed JSON, got nil")
	}
}
