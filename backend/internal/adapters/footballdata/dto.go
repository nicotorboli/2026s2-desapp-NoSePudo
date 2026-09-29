package footballdata

type CompetitionPayload struct {
	Name string `json:"name"`
	Code string `json:"code"`
	ID   int64  `json:"id"`
}

type PlayerPayload struct {
	DateOfBirth *string `json:"dateOfBirth"`
	Nationality *string `json:"nationality"`
	ShirtNumber *int    `json:"shirtNumber"`
	Name        string  `json:"name"`
	Position    string  `json:"position"`
	ID          int64   `json:"id"`
}

type TeamPayload struct {
	Name      string          `json:"name"`
	ShortName string          `json:"shortName"`
	TLA       string          `json:"tla"`
	Squad     []PlayerPayload `json:"squad"`
	ID        int64           `json:"id"`
}

type CompetitionTeamsResponse struct {
	Teams       []TeamPayload      `json:"teams"`
	Competition CompetitionPayload `json:"competition"`
}

type TeamDetailResponse struct {
	Name  string          `json:"name"`
	Squad []PlayerPayload `json:"squad"`
	ID    int64           `json:"id"`
}
