package controller

import (
	"encoding/json"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

// HealthController handles incoming HTTP requests for system health.
type HealthController struct {
	svc service.HealthService
}

// NewHealthController creates a new HealthController with injected service.
func NewHealthController(svc service.HealthService) *HealthController {
	return &HealthController{svc: svc}
}

// HandleHealth handles GET /health requests per Constitution Principles I, III, and IV.
func (c *HealthController) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	healthModel, err := c.svc.CheckHealth(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Explicit DTO conversion via DesdeModelo (Principle IV)
	responseDTO := dto.HealthResponseDTO{}.DesdeModelo(healthModel)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(responseDTO)
}
