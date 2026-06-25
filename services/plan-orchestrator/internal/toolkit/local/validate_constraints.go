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
	return "Validate the generated itinerary against key constraints such as destination consistency, daily activity count, budget limit, and weather adaptation. Args: request, plan, draft, optional weather_summary."
}

func (t *ValidateConstraintsTool) Execute(_ context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
	input, err := parseValidationInput(args)
	if err != nil {
		return domain.ToolExecution{}, err
	}

	issues := make([]string, 0, 4)
	checks := make([]string, 0, 4)

	if issue := validateDestinationConsistency(input.Request, input.Plan, input.Draft); issue != "" {
		issues = append(issues, issue)
	} else {
		checks = append(checks, "destination consistency passed")
	}

	if issue := validateDailyActivityCount(input.Request, input.Plan); issue != "" {
		issues = append(issues, issue)
	} else if len(input.Plan.Days) > 0 {
		checks = append(checks, "daily activity count passed")
	}

	if issue := validateBudget(input.Request, input.Plan, input.Draft); issue != "" {
		issues = append(issues, issue)
	} else {
		checks = append(checks, "budget constraint passed")
	}

	if issue := validateWeatherAdaptation(input.WeatherSummary, input.Plan, input.Draft); issue != "" {
		issues = append(issues, issue)
	} else if strings.TrimSpace(input.WeatherSummary) != "" {
		checks = append(checks, "weather adaptation passed")
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
	Plan           contracts.Plan
	Draft          string
	WeatherSummary string
}

func parseValidationInput(args map[string]interface{}) (validationInput, error) {
	requestValue, ok := args["request"]
	if !ok {
		return validationInput{}, fmt.Errorf("missing request argument")
	}

	req, err := decodeGeneratePlanRequest(requestValue)
	if err != nil {
		return validationInput{}, err
	}

	plan, err := decodePlanArg(args["plan"])
	if err != nil {
		return validationInput{}, err
	}

	draft, _ := args["draft"].(string)
	weatherSummary, _ := args["weather_summary"].(string)

	if strings.TrimSpace(draft) == "" && strings.TrimSpace(plan.Summary) == "" {
		return validationInput{}, fmt.Errorf("draft or plan summary is required")
	}

	return validationInput{
		Request:        req,
		Plan:           plan,
		Draft:          draft,
		WeatherSummary: weatherSummary,
	}, nil
}

func decodePlanArg(value interface{}) (contracts.Plan, error) {
	if value == nil {
		return contracts.Plan{}, fmt.Errorf("missing plan argument")
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return contracts.Plan{}, fmt.Errorf("marshal plan: %w", err)
	}

	var plan contracts.Plan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return contracts.Plan{}, fmt.Errorf("decode plan: %w", err)
	}
	return plan, nil
}

func validateDestinationConsistency(req contracts.GeneratePlanRequest, plan contracts.Plan, draft string) string {
	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		return "user request is missing destination, so destination consistency cannot be validated"
	}

	if strings.TrimSpace(plan.Destination) != "" {
		if strings.TrimSpace(plan.Destination) != destination {
			return fmt.Sprintf("structured plan destination %q does not match requested destination %q", plan.Destination, destination)
		}
		return ""
	}

	if !strings.Contains(draft, destination) {
		return fmt.Sprintf("itinerary draft does not clearly mention requested destination %q", destination)
	}
	return ""
}

func validateDailyActivityCount(req contracts.GeneratePlanRequest, plan contracts.Plan) string {
	if len(plan.Days) == 0 {
		return ""
	}

	maxActivities := extractMaxActivitiesConstraint(req.Constraints)
	if maxActivities <= 0 {
		maxActivities = 2
	}

	violations := make([]string, 0)
	for _, day := range plan.Days {
		count := len(day.Activities)
		if count > maxActivities {
			violations = append(violations, fmt.Sprintf("day %d has %d activities, exceeding limit %d", day.Day, count, maxActivities))
		}
	}

	if len(violations) > 0 {
		return strings.Join(violations, "; ")
	}
	return ""
}

func validateBudget(req contracts.GeneratePlanRequest, plan contracts.Plan, draft string) string {
	budgetLimit := firstNumber(req.Budget)
	if budgetLimit <= 0 {
		return ""
	}

	text := draft
	if strings.TrimSpace(plan.Summary) != "" {
		text = plan.Summary
	}

	draftBudget := extractBudgetFromDraft(text)
	if draftBudget <= 0 {
		return "itinerary output does not provide an explicit budget estimate, so budget cannot be confirmed"
	}

	if draftBudget > budgetLimit {
		return fmt.Sprintf("estimated budget %d exceeds user budget limit %d", draftBudget, budgetLimit)
	}
	return ""
}

func validateWeatherAdaptation(weatherSummary string, plan contracts.Plan, draft string) string {
	if strings.TrimSpace(weatherSummary) == "" {
		return ""
	}

	if !strings.Contains(weatherSummary, "雨") {
		return ""
	}

	if len(plan.Days) > 0 {
		for _, day := range plan.Days {
			for _, activity := range day.Activities {
				if activity.Indoor {
					return ""
				}
				if containsAny(activity.Description, []string{"雨", "室内", "茶馆", "博物馆", "调整"}) {
					return ""
				}
			}
		}
	}

	if containsAny(draft, []string{"室内", "博物馆", "茶馆", "雨具", "调整", "多云", "半户外"}) {
		return ""
	}

	return "weather forecast includes rain, but the itinerary does not clearly show rain-adaptive arrangements"
}

func buildValidationSummary(passed bool, checks, issues []string) string {
	if passed {
		lines := []string{"validation passed"}
		if len(checks) > 0 {
			lines = append(lines, "checks: "+strings.Join(checks, "; "))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{"validation failed"}
	if len(checks) > 0 {
		lines = append(lines, "checks: "+strings.Join(checks, "; "))
	}
	if len(issues) > 0 {
		lines = append(lines, "issues: "+strings.Join(issues, "; "))
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

func extractMaxActivitiesConstraint(constraints []string) int {
	for _, constraint := range constraints {
		if strings.Contains(constraint, "最多") && strings.Contains(constraint, "景点") {
			if value := firstNumber(constraint); value > 0 {
				return value
			}
		}
	}
	return 0
}

func containsAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}
