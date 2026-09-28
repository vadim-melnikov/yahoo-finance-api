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

func TestGetHistoryMalformedJSONOffline(t *testing.T) {
	newOfflineServer(t, []byte(`{"chart": {"result": [`))

	history := newHistory()
	_, err := history.GetHistory("TEST")
	if err == nil {
		t.Fatal("Expected error for malformed JSON, got nil")
	}
}
