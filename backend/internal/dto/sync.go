package dto

type SyncPlayersResponse struct {
	Status           string `json:"status"`
	Message          string `json:"message"`
	TotalProcessed   int    `json:"totalProcessed"`
	TotalUpdated     int    `json:"totalUpdated"`
	TotalDeactivated int    `json:"totalDeactivated"`
}
