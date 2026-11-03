package dto

type SyncStatisticsResponse struct {
	Status         string `json:"status"`
	Message        string `json:"message"`
	TotalProcessed int    `json:"totalProcessed"`
	TotalMatched   int    `json:"totalMatched"`
	TotalUnmatched int    `json:"totalUnmatched"`
	TotalInserted  int    `json:"totalInserted"`
}
