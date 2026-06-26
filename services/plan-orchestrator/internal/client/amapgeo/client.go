package amapgeo

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

type GeocodeResponse struct {
	Status   string    `json:"status"`
	Info     string    `json:"info"`
	InfoCode string    `json:"infocode"`
	Count    string    `json:"count"`
	Geocodes []Geocode `json:"geocodes"`
}

type Geocode struct {
	FormattedAddress string `json:"formatted_address"`
	Country          string `json:"country"`
	Province         string `json:"province"`
	CityCode         string `json:"citycode"`
	City             string `json:"city"`
	District         string `json:"district"`
	Adcode           string `json:"adcode"`
	Location         string `json:"location"`
	Level            string `json:"level"`
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

func (c *Client) Geocode(ctx context.Context, city, address string) (GeocodeResponse, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return GeocodeResponse{}, fmt.Errorf("amap api key is empty")
	}
	if strings.TrimSpace(address) == "" {
		return GeocodeResponse{}, fmt.Errorf("address is empty")
	}

	query := url.Values{}
	query.Set("key", c.apiKey)
	query.Set("address", address)
	if strings.TrimSpace(city) != "" {
		query.Set("city", city)
	}

	endpoint := c.baseURL + "/v3/geocode/geo?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return GeocodeResponse{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GeocodeResponse{}, fmt.Errorf("call amap geocode: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeocodeResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return GeocodeResponse{}, fmt.Errorf("amap status %d: %s", resp.StatusCode, string(body))
	}

	var parsed GeocodeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return GeocodeResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if parsed.Status != "1" {
		return GeocodeResponse{}, fmt.Errorf("amap error %s: %s", parsed.InfoCode, parsed.Info)
	}

	return parsed, nil
}
