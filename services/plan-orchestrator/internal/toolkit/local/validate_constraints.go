package local

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	"github.com/travel-agent/shared/contracts"
)

type ValidateConstraintsTool struct{}

func NewValidateConstraintsTool() *ValidateConstraintsTool {
	return &ValidateConstraintsTool{}
}

func (t *ValidateConstraintsTool) Name() string {
	return "validate_constraints"
}

func (t *ValidateConstraintsTool) Description() string {
	return "Validate the generated itinerary against key constraints such as budget limit, destination consistency, and weather adaptation. Args: request, draft, optional weather_summary."
}

func (t *ValidateConstraintsTool) Execute(_ context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
	input, err := parseValidationInput(args)
	if err != nil {
		return domain.ToolExecution{}, err
	}

	issues := make([]string, 0, 3)
	checks := make([]string, 0, 3)

	if issue := validateDestinationConsistency(input.Request, input.Draft); issue != "" {
		issues = append(issues, issue)
	} else {
		checks = append(checks, "目的地一致性通过")
	}

	if issue := validateBudget(input.Request, input.Draft); issue != "" {
		issues = append(issues, issue)
	} else {
		checks = append(checks, "预算约束通过")
	}

	if issue := validateWeatherAdaptation(input.WeatherSummary, input.Draft); issue != "" {
		issues = append(issues, issue)
	} else if strings.TrimSpace(input.WeatherSummary) != "" {
		checks = append(checks, "天气适配检查通过")
	}

	passed := len(issues) == 0
	summary := buildValidationSummary(passed, checks, issues)

	return domain.ToolExecution{
		Success: true,
		Output:  summary,
		Meta: map[string]interface{}{
			"passed": passed,
			"checks": checks,
			"issues": issues,
		},
	}, nil
}

type validationInput struct {
	Request        contracts.GeneratePlanRequest
	Draft          string
	WeatherSummary string
}

func parseValidationInput(args map[string]interface{}) (validationInput, error) {
	requestValue, ok := args["request"]
	if !ok {
		return validationInput{}, fmt.Errorf("missing request argument")
	}

	requestBytes, err := json.Marshal(requestValue)
	if err != nil {
		return validationInput{}, fmt.Errorf("marshal request: %w", err)
	}

	var req contracts.GeneratePlanRequest
	if err := json.Unmarshal(requestBytes, &req); err != nil {
		return validationInput{}, fmt.Errorf("decode request: %w", err)
	}

	draft, _ := args["draft"].(string)
	weatherSummary, _ := args["weather_summary"].(string)
	if strings.TrimSpace(draft) == "" {
		return validationInput{}, fmt.Errorf("draft is required")
	}

	return validationInput{
		Request:        req,
		Draft:          draft,
		WeatherSummary: weatherSummary,
	}, nil
}

func validateDestinationConsistency(req contracts.GeneratePlanRequest, draft string) string {
	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		return "用户请求中缺少目的地，无法校验目的地一致性。"
	}

	if !strings.Contains(draft, destination) {
		return fmt.Sprintf("行程草案未明确提及目的地“%s”，可能存在目的地不一致问题。", destination)
	}
	return ""
}

func validateBudget(req contracts.GeneratePlanRequest, draft string) string {
	budgetLimit := firstNumber(req.Budget)
	if budgetLimit <= 0 {
		return ""
	}

	draftBudget := extractBudgetFromDraft(draft)
	if draftBudget <= 0 {
		return "行程草案没有给出明确预算估算，无法确认是否超出预算。"
	}

	if draftBudget > budgetLimit {
		return fmt.Sprintf("行程草案估算预算约为%d，超过用户预算上限%d。", draftBudget, budgetLimit)
	}
	return ""
}

func validateWeatherAdaptation(weatherSummary, draft string) string {
	if strings.TrimSpace(weatherSummary) == "" {
		return ""
	}

	rainy := strings.Contains(weatherSummary, "雨")
	if !rainy {
		return ""
	}

	adaptationHints := []string{"室内", "博物馆", "茶馆", "雨具", "调整", "多云", "半户外"}
	for _, hint := range adaptationHints {
		if strings.Contains(draft, hint) {
			return ""
		}
	}

	return "天气预报包含降雨信息，但行程草案没有明显体现雨天适配安排。"
}

func buildValidationSummary(passed bool, checks, issues []string) string {
	if passed {
		lines := []string{"约束校验通过。"}
		if len(checks) > 0 {
			lines = append(lines, "已通过项："+strings.Join(checks, "；"))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{"约束校验未通过。"}
	if len(checks) > 0 {
		lines = append(lines, "已通过项："+strings.Join(checks, "；"))
	}
	if len(issues) > 0 {
		lines = append(lines, "发现问题："+strings.Join(issues, "；"))
	}
	return strings.Join(lines, "\n")
}

func firstNumber(text string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(text)
	if match == "" {
		return 0
	}
	value, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return value
}

func extractBudgetFromDraft(draft string) int {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)total estimated cost[^0-9]*(\d+)`),
		regexp.MustCompile(`(?i)budget[^0-9]*(\d+)`),
		regexp.MustCompile(`总预算[^0-9]*(\d+)`),
		regexp.MustCompile(`预算[^0-9]*(\d+)`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(draft)
		if len(matches) < 2 {
			continue
		}
		value, err := strconv.Atoi(matches[1])
		if err == nil {
			return value
		}
	}
	return 0
}
