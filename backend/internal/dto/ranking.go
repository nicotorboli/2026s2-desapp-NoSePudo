package dto

type PlayerRankingItemResponse struct {
	Score    *float64 `json:"score"`
	Name     string   `json:"name"`
	Club     string   `json:"club"`
	League   string   `json:"league"`
	Position string   `json:"position"`
	ID       int64    `json:"id"`
}

type PlayerRankingPageResponse struct {
	Items  []PlayerRankingItemResponse `json:"items"`
	Offset int                         `json:"offset"`
	Limit  int                         `json:"limit"`
	Total  int                         `json:"total"`
}
