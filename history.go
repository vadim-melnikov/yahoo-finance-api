package yahoofinanceapi

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
)

type YahooHistoryRespose struct {
	Chart YahooChart `json:"chart"`
}

type YahooChart struct {
	Result []YahooHistoryResult `json:"result"`
}

type YahooHistoryResult struct {
	Meta       YahooMeta      `json:"meta"`
	Timestamp  []int64        `json:"timestamp"`
	Events     YahooEvents    `json:"events"`
	Indicators YahooIndicator `json:"indicators"`
}

// YahooEvents holds the corporate events Yahoo returns when HistoryQuery.Events is set.
// Yahoo keys each event by its timestamp.
type YahooEvents struct {
	Splits map[string]YahooSplitEvent `json:"splits"`
}

type YahooSplitEvent struct {
	Date        int64   `json:"date"`
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
	SplitRatio  string  `json:"splitRatio"`
}

// Split is a stock split. A 4-for-1 split has Numerator 4 and Denominator 1,
// a 1-for-10 reverse split has Numerator 1 and Denominator 10.
type Split struct {
	Date        time.Time // moment of the event, UTC
	Numerator   float64
	Denominator float64
	Ratio       string // as Yahoo sent it, e.g. "4:1"
}

type YahooMeta struct {
	Currency             string                 `json:"currency"`
	Symbol               string                 `json:"symbol"`
	ExchangeName         string                 `json:"exchangeName"`
	FullExchangeName     string                 `json:"fullExchangeName"`
	InstrumentType       string                 `json:"instrumentType"`
	FirstTradeDate       int64                  `json:"firstTradeDate"`
	RegularMarketTime    int64                  `json:"regularMarketTime"`
	HasPrePostMarketData bool                   `json:"hasPrePostMarketData"`
	GmtOffset            int                    `json:"gmtoffset"`
	Timezone             string                 `json:"timezone"`
	ExchangeTimezoneName string                 `json:"exchangeTimezoneName"`
	RegularMarketPrice   float64                `json:"regularMarketPrice"`
	FiftyTwoWeekHigh     float64                `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow      float64                `json:"fiftyTwoWeekLow"`
	RegularMarketDayHigh float64                `json:"regularMarketDayHigh"`
	RegularMarketDayLow  float64                `json:"regularMarketDayLow"`
	RegularMarketVolume  int64                  `json:"regularMarketVolume"`
	LongName             string                 `json:"longName"`
	ShortName            string                 `json:"shortName"`
	ChartPreviousClose   float64                `json:"chartPreviousClose"`
	PreviousClose        float64                `json:"previousClose"`
	Scale                int                    `json:"scale"`
	PriceHint            int                    `json:"priceHint"`
	CurrentTradingPeriod YahooTradingPeriod     `json:"currentTradingPeriod"`
	TradingPeriods       [][]YahooTradingPeriod `json:"tradingPeriods"`
	DataGranularity      string                 `json:"dataGranularity"`
	Range                string                 `json:"range"`
	ValidRanges          []string               `json:"validRanges"`
}

type YahooTradingPeriod struct {
	Timezone  string `json:"timezone"`
	End       int64  `json:"end"`
	Start     int64  `json:"start"`
	GmtOffset int    `json:"gmtoffset"`
}

type YahooIndicator struct {
	Quote []YahooQuote `json:"quote"`
}

type YahooQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []int64   `json:"volume"`
}

type PriceData struct {
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   int64
	Currency string
}

type HistoryQuery struct {
	Range     string
	Interval  string
	Start     string
	End       string
	UserAgent string
	// Events asks Yahoo for corporate events along with the prices, e.g. "split" or "div,split".
	// Empty means no events are requested.
	Events string
}

func (hq *HistoryQuery) SetDefault() {
	if hq.Range == "" && hq.Start == "" {
		hq.Range = "1mo"
	}
	if hq.Interval == "" {
		hq.Interval = "1d"
	}
	if hq.Start != "" {
		t, err := time.Parse("2006-01-02", hq.Start)
		if err != nil {
			log.Printf("Failed to parse start date: %v\n", err)
			hq.Start = "default"
		} else {
			hq.Start = fmt.Sprintf("%d", t.Unix())
		}
	}
	if hq.End == "" {
		hq.End = fmt.Sprintf("%d", time.Now().Unix())
	} else {
		t, err := time.Parse("2006-01-02", hq.End)
		if err != nil {
			log.Printf("Failed to parse end date: %v\n", err)
			hq.End = fmt.Sprintf("%d", time.Now().Unix())
		} else {
			hq.End = fmt.Sprintf("%d", t.Unix())
		}
	}
	if hq.UserAgent == "" {
		hq.UserAgent = USER_AGENTS[rand.Intn(len(USER_AGENTS))]
	}
}

type History struct {
	query  *HistoryQuery
	client *Client
}

func newHistory() *History {
	return &History{query: &HistoryQuery{}, client: getClient()}
}

func (h *History) SetQuery(query HistoryQuery) {
	h.query = &query
}

// historyParams builds the v8/finance/chart query parameters from a query
// that has already been through SetDefault.
func historyParams(query *HistoryQuery) url.Values {
	params := url.Values{}
	if query.Range != "" {
		params.Add("range", query.Range)
	}
	params.Add("interval", query.Interval)
	params.Add("period1", query.Start)
	params.Add("period2", query.End)
	if query.Events != "" {
		params.Add("events", query.Events)
	}
	return params
}

// returns the price/volume history of the given symbol as a YahooHistoryResponse
// If you want to adjust the query range change h.query.Range = "6mo" for 6 month
func (h *History) GetHistory(symbol string) (YahooHistoryRespose, error) {
	h.query.SetDefault()
	params := historyParams(h.query)

	endpoint := fmt.Sprintf("%s/v8/finance/chart/%s", BASE_URL, symbol)
	resp, err := h.client.Get(endpoint, params)
	if err != nil {
		slog.Error("Failed to get history", "err", err)
		return YahooHistoryRespose{}, err
	}
	defer resp.Body.Close()

	var historyResponse YahooHistoryRespose
	if err := json.NewDecoder(resp.Body).Decode(&historyResponse); err != nil {
		slog.Error("Failed to decode history data JSON response", "err", err)
		return YahooHistoryRespose{}, fmt.Errorf("failed to decode history data JSON response: %w", err)
	}

	if len(historyResponse.Chart.Result) == 0 {
		return YahooHistoryRespose{}, fmt.Errorf("no data found for symbol: %s", symbol)
	}

	return historyResponse, nil
}

func (h *History) transformData(data YahooHistoryRespose) map[string]PriceData {
	d := make(map[string]PriceData)
	for i, result := range data.Chart.Result[0].Timestamp {
		t := time.Unix(result, 0)
		var key string
		if strings.HasSuffix(h.query.Interval, "d") || strings.HasSuffix(h.query.Interval, "wk") || strings.HasSuffix(h.query.Interval, "mo") {
			key = t.Format("2006-01-02")
		} else {
			key = t.Format("2006-01-02 15:04:05")
		}
		d[key] = PriceData{
			Open:     data.Chart.Result[0].Indicators.Quote[0].Open[i],
			High:     data.Chart.Result[0].Indicators.Quote[0].High[i],
			Low:      data.Chart.Result[0].Indicators.Quote[0].Low[i],
			Close:    data.Chart.Result[0].Indicators.Quote[0].Close[i],
			Volume:   data.Chart.Result[0].Indicators.Quote[0].Volume[i],
			Currency: data.Chart.Result[0].Meta.Currency,
		}
	}
	return d
}

// transformSplits returns the splits of the response in ascending date order.
// A response without events gives an empty slice. A split with a non-positive
// numerator or denominator is an error.
func (h *History) transformSplits(data YahooHistoryRespose) ([]Split, error) {
	events := data.Chart.Result[0].Events.Splits
	splits := make([]Split, 0, len(events))
	for key, event := range events {
		if event.Numerator <= 0 || event.Denominator <= 0 {
			return nil, fmt.Errorf("invalid split %s: numerator %v, denominator %v", key, event.Numerator, event.Denominator)
		}
		splits = append(splits, Split{
			Date:        time.Unix(event.Date, 0).UTC(),
			Numerator:   event.Numerator,
			Denominator: event.Denominator,
			Ratio:       event.SplitRatio,
		})
	}
	sort.Slice(splits, func(i, j int) bool { return splits[i].Date.Before(splits[j].Date) })
	return splits, nil
}
