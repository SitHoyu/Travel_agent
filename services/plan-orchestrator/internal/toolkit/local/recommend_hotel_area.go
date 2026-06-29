package local

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/travel-agent/services/plan-orchestrator/internal/client/amaphotel"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
	"github.com/travel-agent/shared/contracts"
)

type RecommendHotelAreaTool struct {
	hotelClient *amaphotel.Client
}

func NewRecommendHotelAreaTool(hotelClient *amaphotel.Client) *RecommendHotelAreaTool {
	return &RecommendHotelAreaTool{hotelClient: hotelClient}
}

func (t *RecommendHotelAreaTool) Name() string {
	return "recommend_hotel_area"
}

func (t *RecommendHotelAreaTool) Description() string {
	return "Recommend 2-3 hotel areas based on the validated itinerary, destination, budget, and preferences, and attach 0-3 nearby hotel candidates. Args: request, plan."
}

func (t *RecommendHotelAreaTool) Execute(ctx context.Context, args map[string]interface{}) (domain.ToolExecution, error) {
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
	if t.hotelClient != nil && result.RecommendedCenter != nil {
		hotels, err := t.fetchNearbyHotels(ctx, req.Destination, *result.RecommendedCenter)
		if err == nil {
			result.NearbyHotels = hotels
			if len(hotels) == 0 {
				result.NearbyHotelsError = "nearby hotel query succeeded but returned no usable hotel candidates"
			}
		} else {
			result.NearbyHotelsError = err.Error()
		}
	} else if t.hotelClient == nil {
		result.NearbyHotelsError = "hotel client is not configured"
	} else if result.RecommendedCenter == nil {
		result.NearbyHotelsError = "recommended center is empty"
	}

	return domain.ToolExecution{
		Success: true,
		Output:  result.Summary,
		Meta: map[string]interface{}{
			"hotel_areas": result,
		},
	}, nil
}

func (t *RecommendHotelAreaTool) fetchNearbyHotels(ctx context.Context, city string, center contracts.GeoPoint) ([]contracts.HotelCandidate, error) {
	resp, err := t.hotelClient.SearchNearbyHotels(ctx, city, center.Longitude, center.Latitude, 3)
	if err != nil {
		return nil, err
	}

	hotels := make([]contracts.HotelCandidate, 0, len(resp.POIs))
	for _, poi := range resp.POIs {
		lng, lat, err := parsePOILocation(poi.Location)
		if err != nil {
			continue
		}

		hotels = append(hotels, contracts.HotelCandidate{
			Name:      poi.Name,
			Address:   poi.Address,
			DistanceM: parseDistanceMeters(poi.Distance),
			PhotoURL:  firstPhotoURL(poi.Photos),
			Location: contracts.GeoPoint{
				Longitude: lng,
				Latitude:  lat,
			},
		})
	}

	if len(resp.POIs) > 0 && len(hotels) == 0 {
		return nil, fmt.Errorf("amap returned %d pois but none could be converted into hotel candidates", len(resp.POIs))
	}
	return hotels, nil
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

type activityPoint struct {
	Name     string
	AreaName string
	Lng      float64
	Lat      float64
}

func buildHotelAreaRecommendation(req contracts.GeneratePlanRequest, plan contracts.Plan) contracts.HotelAreaRecommendationResult {
	scores := scoreGenericAreas(req, plan)
	if len(scores) == 0 {
		fallbackArea := strings.TrimSpace(req.Destination) + "核心城区"
		return contracts.HotelAreaRecommendationResult{
			Summary: "当前行程区域信息较少，建议优先住在核心城区，方便覆盖主要景点与餐饮区。",
			RecommendedCenter: computeRecommendedCenter(plan, []contracts.HotelAreaRecommendation{
				{Area: fallbackArea, Priority: 1},
			}),
			Recommendations: []contracts.HotelAreaRecommendation{
				{
					Area:        fallbackArea,
					Priority:    1,
					PriceRange:  estimateHotelPriceRange(req.Budget, maxInt(1, len(plan.Days))),
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

	limit := minInt(3, len(scores))
	recommendations := make([]contracts.HotelAreaRecommendation, 0, limit)
	for i := 0; i < limit; i++ {
		item := scores[i]
		recommendations = append(recommendations, contracts.HotelAreaRecommendation{
			Area:        item.Name,
			Priority:    i + 1,
			PriceRange:  estimateHotelPriceRange(req.Budget, maxInt(1, len(plan.Days))),
			FitReason:   buildGenericAreaFitReason(item, req),
			Pros:        buildGenericPros(item, req),
			Cons:        buildGenericCons(item),
			SuitableFor: buildGenericSuitableFor(req.Preferences),
		})
	}

	return contracts.HotelAreaRecommendationResult{
		Summary:           buildGenericHotelAreaSummary(recommendations, req),
		RecommendedCenter: computeRecommendedCenter(plan, recommendations),
		Recommendations:   recommendations,
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

func computeRecommendedCenter(plan contracts.Plan, recommendations []contracts.HotelAreaRecommendation) *contracts.GeoPoint {
	allPoints := collectActivityPoints(plan, "")
	if len(allPoints) == 0 {
		return nil
	}

	if len(recommendations) > 0 {
		topArea := strings.TrimSpace(recommendations[0].Area)
		areaPoints := collectActivityPoints(plan, topArea)
		if point := pickRepresentativePoint(areaPoints); point != nil {
			return &contracts.GeoPoint{
				Longitude: point.Lng,
				Latitude:  point.Lat,
				Source:    "top_area_representative_point",
			}
		}
		if point := centroidPoint(areaPoints); point != nil {
			return &contracts.GeoPoint{
				Longitude: point.Lng,
				Latitude:  point.Lat,
				Source:    "top_area_centroid",
			}
		}
	}

	if point := pickRepresentativePoint(allPoints); point != nil {
		return &contracts.GeoPoint{
			Longitude: point.Lng,
			Latitude:  point.Lat,
			Source:    "all_points_representative_point",
		}
	}

	if point := centroidPoint(allPoints); point != nil {
		return &contracts.GeoPoint{
			Longitude: point.Lng,
			Latitude:  point.Lat,
			Source:    "all_points_centroid",
		}
	}

	return nil
}

func collectActivityPoints(plan contracts.Plan, areaFilter string) []activityPoint {
	points := make([]activityPoint, 0)
	for _, day := range plan.Days {
		for _, activity := range day.Activities {
			if activity.Longitude == 0 && activity.Latitude == 0 {
				continue
			}

			areaName := firstAreaCandidate(activity)
			if strings.TrimSpace(areaFilter) != "" && areaName != strings.TrimSpace(areaFilter) {
				continue
			}

			points = append(points, activityPoint{
				Name:     activity.Name,
				AreaName: areaName,
				Lng:      activity.Longitude,
				Lat:      activity.Latitude,
			})
		}
	}
	return points
}

func firstAreaCandidate(activity contracts.Activity) string {
	candidates := extractAreaCandidates("", activity)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func pickRepresentativePoint(points []activityPoint) *activityPoint {
	if len(points) == 0 {
		return nil
	}
	if len(points) == 1 {
		point := points[0]
		return &point
	}

	bestIndex := 0
	bestScore := math.MaxFloat64
	for i := range points {
		total := 0.0
		for j := range points {
			if i == j {
				continue
			}
			total += squaredDistance(points[i], points[j])
		}
		if total < bestScore {
			bestScore = total
			bestIndex = i
		}
	}

	point := points[bestIndex]
	return &point
}

func centroidPoint(points []activityPoint) *activityPoint {
	if len(points) == 0 {
		return nil
	}

	var sumLng float64
	var sumLat float64
	for _, point := range points {
		sumLng += point.Lng
		sumLat += point.Lat
	}

	return &activityPoint{
		Lng: sumLng / float64(len(points)),
		Lat: sumLat / float64(len(points)),
	}
}

func squaredDistance(a, b activityPoint) float64 {
	dLng := a.Lng - b.Lng
	dLat := a.Lat - b.Lat
	return dLng*dLng + dLat*dLat
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

	if strings.TrimSpace(destination) != "" {
		text = strings.TrimPrefix(text, strings.TrimSpace(destination))
	}
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
	text = strings.Trim(text, "- ,，。；;")
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

func buildGenericCons(item genericAreaScore) []string {
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

func parsePOILocation(location string) (float64, float64, error) {
	parts := strings.Split(strings.TrimSpace(location), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid poi location %q", location)
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, err
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, err
	}
	return lng, lat, nil
}

func parseDistanceMeters(distance string) int {
	value, err := strconv.Atoi(strings.TrimSpace(distance))
	if err != nil {
		return 0
	}
	return value
}

func firstPhotoURL(photos []amaphotel.Photo) string {
	if len(photos) == 0 {
		return ""
	}
	return strings.TrimSpace(photos[0].URL)
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
