package yahoofinanceapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
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

func TestHistoryWithSplitsOffline(t *testing.T) {
	lastQuery := newOfflineServer(t, readFixture(t, "chart_with_splits.json"))

	ticker := NewTicker("TEST")
	prices, splits, err := ticker.HistoryWithSplits(HistoryQuery{Start: "2020-08-01", End: "2021-01-31"})
	if err != nil {
		t.Fatalf("HistoryWithSplits returned error: %v", err)
	}
	if got := lastQuery().Get("events"); got != "split" {
		t.Errorf("Expected events parameter %q, got %q", "split", got)
	}
	if len(prices) != 3 {
		t.Errorf("Expected 3 prices, got %d", len(prices))
	}

	want := []Split{
		{Date: time.Date(2020, 8, 31, 13, 30, 0, 0, time.UTC), Numerator: 4, Denominator: 1, Ratio: "4:1"},
		{Date: time.Date(2021, 1, 4, 14, 30, 0, 0, time.UTC), Numerator: 1, Denominator: 10, Ratio: "1:10"},
	}
	if len(splits) != len(want) {
		t.Fatalf("Expected %d splits, got %d: %+v", len(want), len(splits), splits)
	}
	for i, s := range splits {
		if !s.Date.Equal(want[i].Date) || s.Numerator != want[i].Numerator ||
			s.Denominator != want[i].Denominator || s.Ratio != want[i].Ratio {
			t.Errorf("Split %d: expected %+v, got %+v", i, want[i], s)
		}
		if s.Date.Location() != time.UTC {
			t.Errorf("Split %d: expected Date in UTC, got %v", i, s.Date.Location())
		}
	}
}

func TestHistoryWithSplitsNoEventsOffline(t *testing.T) {
	newOfflineServer(t, readFixture(t, "chart_no_events.json"))

	ticker := NewTicker("TEST")
	prices, splits, err := ticker.HistoryWithSplits(HistoryQuery{Start: "2020-08-01", End: "2020-09-30"})
	if err != nil {
		t.Fatalf("HistoryWithSplits returned error: %v", err)
	}
	if splits == nil || len(splits) != 0 {
		t.Errorf("Expected an empty, non-nil slice of splits, got %#v", splits)
	}
	if len(prices) != 2 {
		t.Errorf("Expected 2 prices, got %d", len(prices))
	}
}

func TestHistoryWithSplitsZeroDenominatorOffline(t *testing.T) {
	newOfflineServer(t, readFixture(t, "chart_zero_denominator.json"))

	ticker := NewTicker("TEST")
	_, _, err := ticker.HistoryWithSplits(HistoryQuery{Start: "2020-08-01", End: "2020-09-30"})
	if err == nil {
		t.Error("Expected error for a split with a zero denominator, got nil")
	}
}

func TestHistoryDoesNotRequestEventsOffline(t *testing.T) {
	lastQuery := newOfflineServer(t, readFixture(t, "chart_with_splits.json"))

	ticker := NewTicker("TEST")
	if _, _, err := ticker.HistoryWithSplits(HistoryQuery{Start: "2020-08-01", End: "2021-01-31"}); err != nil {
		t.Fatalf("HistoryWithSplits returned error: %v", err)
	}
	if _, err := ticker.History(HistoryQuery{Start: "2020-08-01", End: "2021-01-31"}); err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if lastQuery().Has("events") {
		t.Errorf("Expected History to send no events parameter, got %q", lastQuery().Get("events"))
	}
}

func TestTransformSplitsRejectsZeroParts(t *testing.T) {
	events := []YahooSplitEvent{
		{Date: 1598880600, Numerator: 0, Denominator: 1, SplitRatio: "0:1"},
		{Date: 1598880600, Numerator: 4, Denominator: 0, SplitRatio: "4:0"},
	}
	history := newHistory()
	for _, event := range events {
		data := YahooHistoryRespose{Chart: YahooChart{Result: []YahooHistoryResult{{
			Events: YahooEvents{Splits: map[string]YahooSplitEvent{"1598880600": event}},
		}}}}
		if _, err := history.transformSplits(data); err == nil {
			t.Errorf("Expected error for split %q, got nil", event.SplitRatio)
		}
	}
}
