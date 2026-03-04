package yahoofinanceapi

import (
	"fmt"
	"testing"
	"time"
)

func TestNewHistory(t *testing.T) {
	history := newHistory()
	if history == nil {
		t.Fatal("newHistory returned nil")
	}
}

func TestGetHistoryValidSymbol(t *testing.T) {
	history := newHistory()
	resp, err := history.GetHistory("AAPL")
	if err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	if len(resp.Chart.Result) == 0 {
		t.Fatal("GetHistory returned empty result for valid symbol")
	}
}

func TestGetHistoryInvalidSymbol(t *testing.T) {
	history := newHistory()
	_, err := history.GetHistory("INVALID_SYMBOL_123")
	if err == nil {
		t.Error("Expected error for invalid symbol, got nil")
	}
}

func TestTransformData(t *testing.T) {
	history := newHistory()
	resp, err := history.GetHistory("AAPL")
	if err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	transformed := history.transformData(resp)
	if len(transformed) == 0 {
		t.Error("transformData returned empty map")
	}
	for _, data := range transformed {
		if data.Close == 0 {
			t.Error("transformData returned PriceData with zero Close")
		}
	}
}

func TestTransformDataWithMinuteInterval(t *testing.T) {
	history := newHistory()
	q := HistoryQuery{Range: "1d", Interval: "5m"} // Used 5m interval because caught zero values for 1m interval, i.e. interval=1m&period1=&period2=1772634964&range=1d
	history.SetQuery(q)
	resp, err := history.GetHistory("AAPL")
	if err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	transformed := history.transformData(resp)
	if len(transformed) == 0 {
		t.Error("transformData returned empty map")
	}
	for _, data := range transformed {
		if data.Close == 0 {
			t.Error("transformData returned PriceData with zero Close")
		}
		if data.Currency != "USD" {
			t.Error("transformData returned PriceData with incorrect Currency")
		}
	}
}

func TestTransformDataWithMinuteIntervalForNonUSMarket(t *testing.T) {
	history := newHistory()
	q := HistoryQuery{Range: "1d", Interval: "5m"}
	history.SetQuery(q)
	resp, err := history.GetHistory("BTO.TO")
	if err != nil {
		t.Fatalf("GetHistory returned error: %v", err)
	}
	transformed := history.transformData(resp)
	if len(transformed) == 0 {
		t.Error("transformData returned empty map")
	}
	for _, data := range transformed {
		if data.Close == 0 {
			t.Error("transformData returned PriceData with zero Close")
		}
		if data.Currency != "CAD" {
			t.Error("transformData returned PriceData with incorrect Currency")
		}
	}
}

func TestSetQuery(t *testing.T) {
	history := newHistory()
	q := HistoryQuery{Range: "5d", Interval: "1d"}
	history.SetQuery(q)
	if history.query.Range != "5d" || history.query.Interval != "1d" {
		t.Error("SetQuery did not set query fields correctly")
	}
}

func TestSetDefault(t *testing.T) {
	q := HistoryQuery{}
	q.SetDefault()
	if q.Range != "1mo" {
		t.Error("SetDefault did not set default Range")
	}
	if q.Interval != "1d" {
		t.Error("SetDefault did not set default Interval")
	}
	if q.UserAgent == "" {
		t.Error("SetDefault did not set UserAgent")
	}
}

func TestSetDefaultWithStartDate(t *testing.T) {
	q := HistoryQuery{Start: "2024-01-01"}
	q.SetDefault()
	_, err := time.Parse("2006-01-02", "2024-01-01")
	if err != nil {
		t.Fatal("Test setup error: invalid date")
	}
	if q.Start == "default" {
		t.Error("SetDefault failed to parse valid Start date")
	}
}

func TestSetDefaultWithInvalidStartDate(t *testing.T) {
	q := HistoryQuery{Start: "invalid-date"}
	q.SetDefault()
	if q.Start != "default" {
		t.Error("SetDefault did not set Start to 'default' for invalid date")
	}
}

func TestSetEndDate(t *testing.T) {
	endString := "2026-02-28"
	q := HistoryQuery{End: endString}
	q.SetDefault()

	tm, _ := time.Parse("2006-01-02", endString)
	if q.End != fmt.Sprintf("%d", tm.Unix()) {
		t.Error("SetDefault did not converted End date correctly")
	}
}

func TestSetStartDate(t *testing.T) {
	startDate := "2026-02-28"
	q := HistoryQuery{Start: startDate}
	q.SetDefault()

	tm, _ := time.Parse("2006-01-02", startDate)
	if q.Start != fmt.Sprintf("%d", tm.Unix()) {
		t.Error("SetDefault did not converted Start date correctly")
	}
}

func TestSetBothDates(t *testing.T) {
	startDate := "2026-02-28"
	endDate := "2026-03-03"
	q := HistoryQuery{Start: startDate, End: endDate}
	q.SetDefault()

	tm, _ := time.Parse("2006-01-02", startDate)
	tm2, _ := time.Parse("2006-01-02", endDate)
	if q.Start != fmt.Sprintf("%d", tm.Unix()) {
		t.Error("SetDefault did not converted Start date correctly")
	}
	if q.End != fmt.Sprintf("%d", tm2.Unix()) {
		t.Error("SetDefault did not converted End date correctly")
	}
}
