package local

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/travel-agent/services/plan-orchestrator/internal/client/amapweather"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	// "github.com/travel-agent/shared/contracts"
)

type WeatherTool struct {
	client   *amapweather.Client
	resolver *CityCodeResolver
}

func NewWeatherTool(client *amapweather.Client, resolver *CityCodeResolver) *WeatherTool {
	return &WeatherTool{
		client:   client,
		resolver: resolver,
	}
}

func (t *WeatherTool) Name() string {
	return "query_weather"
}

func (t *WeatherTool) Description() string {
	return "Query the AMap 3-day weather forecast for a destination city. Args: city, start_date, end_date. Supports passing the full travel request as request."
}

func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
	params, err := t.parseArgs(args)
	if err != nil {
		return domain.ToolExecution{}, err
	}

	adcode, ok := t.resolver.Resolve(params.City)
	if !ok {
		return domain.ToolExecution{}, fmt.Errorf("city %s not found in AMap adcode table", params.City)
	}

	resp, err := t.client.Forecast(ctx, adcode)
	if err != nil {
		return domain.ToolExecution{}, err
	}
	if len(resp.Forecasts) == 0 {
		return domain.ToolExecution{}, fmt.Errorf("no forecast returned for city %s", params.City)
	}

	cityForecast := resp.Forecasts[0]
	casts := filterCasts(cityForecast.Casts, params.StartDate, params.EndDate)
	if len(casts) == 0 {
		casts = cityForecast.Casts
	}

	summary := renderWeatherSummary(cityForecast, casts)
	return domain.ToolExecution{
		Success: true,
		Output:  summary,
		Meta: map[string]interface{}{
			"city":       cityForecast.City,
			"province":   cityForecast.Province,
			"adcode":     cityForecast.Adcode,
			"reporttime": cityForecast.ReportTime, //暂时设为三天
			"casts":      casts,
		},
	}, nil
}

type weatherQueryArgs struct {
	City      string
	StartDate string
	EndDate   string
}

func (t *WeatherTool) parseArgs(args map[string]interface{}) (weatherQueryArgs, error) {
	if requestValue, ok := args["request"]; ok {
		req, err := decodeGeneratePlanRequest(requestValue)
		if err != nil {
			return weatherQueryArgs{}, fmt.Errorf("decode request argument: %w", err)
		}
		return weatherQueryArgs{
			City:      req.Destination,
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
		}, nil
	}

	city, _ := args["city"].(string)
	startDate, _ := args["start_date"].(string)
	endDate, _ := args["end_date"].(string)
	if strings.TrimSpace(city) == "" {
		return weatherQueryArgs{}, fmt.Errorf("city is required")
	}
	return weatherQueryArgs{
		City:      city,
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}

func filterCasts(casts []amapweather.DailyCast, startDate, endDate string) []amapweather.DailyCast {
	if strings.TrimSpace(startDate) == "" || strings.TrimSpace(endDate) == "" {
		return casts
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return casts
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return casts
	}

	filtered := make([]amapweather.DailyCast, 0, len(casts))
	for _, cast := range casts {
		date, err := time.Parse("2006-01-02", cast.Date)
		if err != nil {
			continue
		}
		if (date.Equal(start) || date.After(start)) && (date.Equal(end) || date.Before(end)) {
			filtered = append(filtered, cast)
		}
	}
	return filtered
}

func renderWeatherSummary(forecast amapweather.CityForecast, casts []amapweather.DailyCast) string {
	lines := []string{
		fmt.Sprintf("%s未来天气预报（报告时间：%s）", forecast.City, forecast.ReportTime),
	}
	for _, cast := range casts {
		lines = append(lines,
			fmt.Sprintf("%s：白天%s，夜间%s，气温%s-%s℃，白天风向%s，风力%s。",
				cast.Date, cast.DayWeather, cast.NightWeather, cast.NightTemp, cast.DayTemp, cast.DayWind, cast.DayPower),
		)
	}
	return strings.Join(lines, "\n")
}
