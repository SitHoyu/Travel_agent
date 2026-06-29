package contracts

// Plan is the cross-service contract for a generated itinerary.
type Plan struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	Destination string    `json:"destination"`
	Days        []PlanDay `json:"days"`
	Summary     string    `json:"summary"`
}

type HotelAreaRecommendationResult struct {
	Summary           string                    `json:"summary"`
	RecommendedCenter *GeoPoint                 `json:"recommended_center,omitempty"`
	NearbyHotels      []HotelCandidate          `json:"nearby_hotels,omitempty"`
	NearbyHotelsError string                    `json:"nearby_hotels_error,omitempty"`
	Recommendations   []HotelAreaRecommendation `json:"recommendations"`
}

type HotelAreaRecommendation struct {
	Area        string   `json:"area"`
	Priority    int      `json:"priority"`
	PriceRange  string   `json:"price_range"`
	FitReason   string   `json:"fit_reason"`
	Pros        []string `json:"pros,omitempty"`
	Cons        []string `json:"cons,omitempty"`
	SuitableFor []string `json:"suitable_for,omitempty"`
}

type GeoPoint struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Source    string  `json:"source,omitempty"`
}

type HotelCandidate struct {
	Name      string   `json:"name"`
	Address   string   `json:"address,omitempty"`
	DistanceM int      `json:"distance_m,omitempty"`
	PhotoURL  string   `json:"photo_url,omitempty"`
	Location  GeoPoint `json:"location"`
}

type PlanDay struct {
	Day        int        `json:"day"`
	Date       string     `json:"date,omitempty"`
	Theme      string     `json:"theme"`
	Activities []Activity `json:"activities"`
}

type Activity struct {
	Name            string  `json:"name"`
	Location        string  `json:"location"`
	TimeSlot        string  `json:"time_slot"`
	Type            string  `json:"type"`
	Indoor          bool    `json:"indoor"`
	Description     string  `json:"description"`
	ResolvedAddress string  `json:"resolved_address,omitempty"`
	Longitude       float64 `json:"longitude,omitempty"`
	Latitude        float64 `json:"latitude,omitempty"`
	Adcode          string  `json:"adcode,omitempty"`
	GeoLevel        string  `json:"geo_level,omitempty"`
}
