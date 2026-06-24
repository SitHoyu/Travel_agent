package amapweather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type ForecastResponse struct {
	Status    string         `json:"status"`
	Count     string         `json:"count"`
	Info      string         `json:"info"`
	InfoCode  string         `json:"infocode"`
	Forecasts []CityForecast `json:"forecasts"`
}

type CityForecast struct {
	City       string        `json:"city"`
	Adcode     string        `json:"adcode"`
	Province   string        `json:"province"`
	ReportTime string        `json:"reporttime"`
	Casts      []DailyCast   `json:"casts"`
}

type DailyCast struct {
	Date           string `json:"date"`
	Week           string `json:"week"`
	DayWeather     string `json:"dayweather"`
	NightWeather   string `json:"nightweather"`
	DayTemp        string `json:"daytemp"`
	NightTemp      string `json:"nighttemp"`
	DayWind        string `json:"daywind"`
	NightWind      string `json:"nightwind"`
	DayPower       string `json:"daypower"`
	NightPower     string `json:"nightpower"`
	DayTempFloat   string `json:"daytemp_float"`
	NightTempFloat string `json:"nighttemp_float"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) Forecast(ctx context.Context, adcode string) (ForecastResponse, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return ForecastResponse{}, fmt.Errorf("amap api key is empty")
	}
	if strings.TrimSpace(adcode) == "" {
		return ForecastResponse{}, fmt.Errorf("city adcode is empty")
	}

	query := url.Values{}
	query.Set("key", c.apiKey)
	query.Set("city", adcode)
	query.Set("extensions", "all")

	endpoint := c.baseURL + "/v3/weather/weatherInfo?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ForecastResponse{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ForecastResponse{}, fmt.Errorf("call amap weather: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ForecastResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return ForecastResponse{}, fmt.Errorf("amap status %d: %s", resp.StatusCode, string(body))
	}

	var parsed ForecastResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ForecastResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if parsed.Status != "1" {
		return ForecastResponse{}, fmt.Errorf("amap error %s: %s", parsed.InfoCode, parsed.Info)
	}

	return parsed, nil
}
