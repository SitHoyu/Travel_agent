package contracts

// Plan is the cross-service contract for a generated itinerary.
type Plan struct {
	ID          string      `json:"id"`
	Status      string      `json:"status"`
	Title       string      `json:"title"`
	Destination string      `json:"destination"`
	Days        []PlanDay   `json:"days"`
	Summary     string      `json:"summary"`
}

type PlanDay struct {
	Day        int        `json:"day"`
	Date       string     `json:"date,omitempty"`
	Theme      string     `json:"theme"`
	Activities []Activity `json:"activities"`
}

type Activity struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	TimeSlot    string `json:"time_slot"`
	Type        string `json:"type"`
	Indoor      bool   `json:"indoor"`
	Description string `json:"description"`
}
