package local

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	"github.com/travel-agent/shared/contracts"
)

type RecommendHotelAreaTool struct{}

func NewRecommendHotelAreaTool() *RecommendHotelAreaTool {
	return &RecommendHotelAreaTool{}
}

func (t *RecommendHotelAreaTool) Name() string {
	return "recommend_hotel_area"
}

func (t *RecommendHotelAreaTool) Description() string {
	return "Recommend 2-3 hotel areas based on the validated itinerary, destination, budget, and preferences. Args: request, plan."
}

func (t *RecommendHotelAreaTool) Execute(_ context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
	reqValue, ok := args["request"]
	if !ok {
		return domain.ToolExecution{}, fmt.Errorf("missing request argument")
	}

	req, err := decodeGeneratePlanRequest(reqValue)
	if err != nil {
		return domain.ToolExecution{}, err
	}

	plan, err := decodeHotelPlanArg(args["plan"])
	if err != nil {
		return domain.ToolExecution{}, err
	}

	result := buildHotelAreaRecommendation(req, plan)
	return domain.ToolExecution{
		Success: true,
		Output:  result.Summary,
		Meta: map[string]interface{}{
			"hotel_areas": result,
		},
	}, nil
}

func decodeHotelPlanArg(value interface{}) (contracts.Plan, error) {
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

type genericAreaScore struct {
	Name         string
	Score        int
	ActivityHits int
	Keywords     []string
}

func buildHotelAreaRecommendation(req contracts.GeneratePlanRequest, plan contracts.Plan) contracts.HotelAreaRecommendationResult {
	scores := scoreGenericAreas(req, plan)
	if len(scores) == 0 {
		fallbackArea := strings.TrimSpace(req.Destination) + "核心城区"
		return contracts.HotelAreaRecommendationResult{
			Summary: fmt.Sprintf("当前行程区域信息较少，建议优先住在%s，方便覆盖主要景点与餐饮区。", fallbackArea),
			Recommendations: []contracts.HotelAreaRecommendation{
				{
					Area:        fallbackArea,
					Priority:    1,
					PriceRange:  estimateHotelPriceRange(req.Budget, max(1, len(plan.Days))),
					FitReason:   "当前活动区域信息不足，先住核心城区更稳妥，后续可按景点分布再细化。",
					Pros:        []string{"通用性强", "更容易覆盖主要景点"},
					Cons:        []string{"未针对具体活动片区做精细优化"},
					SuitableFor: []string{"首次规划", "信息不足时的稳妥方案"},
				},
			},
		}
	}

	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return scores[i].ActivityHits > scores[j].ActivityHits
		}
		return scores[i].Score > scores[j].Score
	})

	limit := min(3, len(scores))
	recommendations := make([]contracts.HotelAreaRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		item := scores[i]
		recommendations = append(recommendations, contracts.HotelAreaRecommendation{
			Area:        item.Name,
			Priority:    i + 1,
			PriceRange:  estimateHotelPriceRange(req.Budget, max(1, len(plan.Days))),
			FitReason:   buildGenericAreaFitReason(item, req),
			Pros:        buildGenericPros(item, req),
			Cons:        buildGenericCons(item, req),
			SuitableFor: buildGenericSuitableFor(req.Preferences),
		})
	}

	return contracts.HotelAreaRecommendationResult{
		Summary:         buildGenericHotelAreaSummary(recommendations, req),
		Recommendations: recommendations,
	}
}

func scoreGenericAreas(req contracts.GeneratePlanRequest, plan contracts.Plan) []genericAreaScore {
	if len(plan.Days) == 0 {
		return nil
	}

	scoreMap := make(map[string]*genericAreaScore)
	for _, day := range plan.Days {
		for _, activity := range day.Activities {
			areaNames := extractAreaCandidates(req.Destination, activity)
			for _, areaName := range areaNames {
				entry := scoreMap[areaName]
				if entry == nil {
					entry = &genericAreaScore{Name: areaName}
					scoreMap[areaName] = entry
				}

				entry.Score += 3
				entry.ActivityHits++
				entry.Keywords = appendUnique(entry.Keywords, firstNonEmpty(activity.Location, activity.Name))

				if activity.Indoor {
					entry.Score++
				}
				if activity.Type == "food" {
					entry.Score++
				}
			}
		}
	}

	applyGenericPreferenceBoosts(req.Preferences, scoreMap)
	applyGenericBudgetAdjustments(req.Budget, scoreMap)

	result := make([]genericAreaScore, 0, len(scoreMap))
	for _, item := range scoreMap {
		if item.Score <= 0 {
			continue
		}
		result = append(result, *item)
	}
	return result
}

func extractAreaCandidates(destination string, activity contracts.Activity) []string {
	candidates := make([]string, 0, 3)

	if district := extractDistrictLikeName(activity.ResolvedAddress); district != "" {
		candidates = append(candidates, district)
	}
	if localArea := extractLocationArea(activity.Location, destination); localArea != "" {
		candidates = append(candidates, localArea)
	}
	if district := extractDistrictLikeName(activity.Location); district != "" {
		candidates = append(candidates, district)
	}

	return uniqueStrings(candidates)
}

func extractDistrictLikeName(text string) string {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return ""
	}

	segments := splitAreaSegments(normalized)
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if hasAreaSuffix(segment) {
			return segment
		}
	}
	return ""
}

func extractLocationArea(location, destination string) string {
	text := strings.TrimSpace(location)
	if text == "" {
		return ""
	}

	text = strings.TrimPrefix(text, strings.TrimSpace(destination))
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	if district := extractDistrictLikeName(text); district != "" {
		return district
	}

	runes := []rune(text)
	if len(runes) > 8 {
		text = string(runes[:8])
	}
	text = strings.Trim(text, "- ,，。;；")
	if text == "" {
		return ""
	}
	return text + "周边"
}

func splitAreaSegments(text string) []string {
	replacer := strings.NewReplacer(
		",", " ",
		"，", " ",
		"/", " ",
		"·", " ",
		"-", " ",
	)
	return strings.Fields(replacer.Replace(text))
}

func hasAreaSuffix(text string) bool {
	suffixes := []string{"区", "县", "市", "镇", "乡", "村", "街道", "新区", "开发区", "园区"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(text, suffix) {
			return true
		}
	}
	return false
}

func applyGenericPreferenceBoosts(preferences []string, scores map[string]*genericAreaScore) {
	joined := strings.Join(preferences, " ")
	for _, item := range scores {
		if strings.Contains(joined, "美食") && containsAny(item.Name, []string{"老城", "古城", "城区", "街", "路", "商圈"}) {
			item.Score += 2
		}
		if strings.Contains(joined, "慢节奏") && containsAny(item.Name, []string{"湖", "山", "村", "镇", "景区"}) {
			item.Score += 2
		}
		if containsAny(joined, item.Keywords) {
			item.Score += 2
		}
	}
}

func applyGenericBudgetAdjustments(budget string, scores map[string]*genericAreaScore) {
	value := firstNumber(budget)
	for _, item := range scores {
		switch {
		case value > 0 && value <= 2000:
			if containsAny(item.Name, []string{"城区", "中心", "广场", "街道"}) {
				item.Score += 1
			}
		case value > 2000 && value <= 4000:
			item.Score += 1
		default:
			if containsAny(item.Name, []string{"景区", "湖", "山", "村"}) {
				item.Score += 1
			}
		}
	}
}

func buildGenericAreaFitReason(item genericAreaScore, req contracts.GeneratePlanRequest) string {
	reasons := item.Keywords
	if len(reasons) > 3 {
		reasons = reasons[:3]
	}
	if len(reasons) == 0 {
		return fmt.Sprintf("该区域与%s当前行程分布更接近，适合作为住宿落点。", req.Destination)
	}
	return fmt.Sprintf("当前行程多次涉及%s，住在%s可减少往返通勤。", strings.Join(reasons, "、"), item.Name)
}

func buildGenericPros(item genericAreaScore, req contracts.GeneratePlanRequest) []string {
	pros := []string{
		"更贴近当前主要活动分布",
		"便于覆盖行程中的核心点位",
	}
	if strings.Contains(strings.Join(req.Preferences, " "), "美食") {
		pros = append(pros, "更方便衔接当地餐饮与夜间活动")
	}
	return pros
}

func buildGenericCons(item genericAreaScore, req contracts.GeneratePlanRequest) []string {
	cons := []string{
		"仍需结合实际酒店库存和交通情况二次确认",
	}
	if containsAny(item.Name, []string{"景区", "湖", "山", "村"}) {
		cons = append(cons, "热门景点周边价格可能在旺季波动更明显")
	}
	return cons
}

func buildGenericSuitableFor(preferences []string) []string {
	if len(preferences) == 0 {
		return []string{"通用型出行需求"}
	}
	if len(preferences) > 3 {
		return preferences[:3]
	}
	return preferences
}

func buildGenericHotelAreaSummary(recommendations []contracts.HotelAreaRecommendation, req contracts.GeneratePlanRequest) string {
	if len(recommendations) == 0 {
		return "暂未生成明确的住宿区域建议。"
	}

	names := make([]string, 0, len(recommendations))
	for _, item := range recommendations {
		names = append(names, item.Area)
	}
	return fmt.Sprintf("结合%s的行程分布、预算和偏好，优先推荐住在%s。", req.Destination, strings.Join(names, "或"))
}

func estimateHotelPriceRange(budget string, days int) string {
	value := firstNumber(budget)
	if value <= 0 || days <= 0 {
		return "300-500 RMB/night"
	}

	nightly := value / days / 3
	switch {
	case nightly <= 250:
		return "200-300 RMB/night"
	case nightly <= 450:
		return "300-500 RMB/night"
	case nightly <= 700:
		return "500-700 RMB/night"
	default:
		return "700+ RMB/night"
	}
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
