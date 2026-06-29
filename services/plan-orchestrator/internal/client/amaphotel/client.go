package amaphotel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type AroundResponse struct {
	Status     string      `json:"status"`
	Info       string      `json:"info"`
	InfoCode   string      `json:"infocode"`
	Count      string      `json:"count"`
	Suggestion interface{} `json:"suggestion"`
	POIs       []POI       `json:"pois"`
}

type POI struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Distance string  `json:"distance"`
	Location string  `json:"location"`
	Type     string  `json:"type"`
	TypeCode string  `json:"typecode"`
	AdName   string  `json:"adname"`
	CityName string  `json:"cityname"`
	BizType  string  `json:"biz_type"`
	Photos   []Photo `json:"photos"`
}

type Photo struct {
	URL string `json:"url"`
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

func (c *Client) SearchNearbyHotels(ctx context.Context, city string, longitude, latitude float64, limit int) (AroundResponse, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return AroundResponse{}, fmt.Errorf("amap api key is empty")
	}
	if limit <= 0 {
		limit = 3
	}

	query := url.Values{}
	query.Set("key", c.apiKey)
	query.Set("city", city)
	query.Set("location", fmt.Sprintf("%.6f,%.6f", longitude, latitude))
	query.Set("types", "10")
	query.Set("keywords", "酒店")
	query.Set("offset", strconv.Itoa(limit))

	endpoint := c.baseURL + "/v3/place/around?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return AroundResponse{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AroundResponse{}, fmt.Errorf("call amap place around: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AroundResponse{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return AroundResponse{}, fmt.Errorf("amap status %d: %s", resp.StatusCode, string(body))
	}

	var parsed AroundResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return AroundResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if parsed.Status != "1" {
		return AroundResponse{}, fmt.Errorf("amap error %s: %s", parsed.InfoCode, parsed.Info)
	}

	return parsed, nil
}
