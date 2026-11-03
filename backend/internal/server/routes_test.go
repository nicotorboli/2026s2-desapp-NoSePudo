package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type stubController struct{}

func (stubController) endpoint() httphandler.Endpoint {
	return func(http.ResponseWriter, *http.Request) error { return nil }
}

type stubPlayerService struct{}

func (stubPlayerService) ListPlayers(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error) {
	return dto.PaginatedResponse[dto.PlayerListItemResponse]{}, nil
}

func (stubPlayerService) GetPlayerByID(ctx context.Context, id int64) (dto.PlayerDetailResponse, error) {
	return dto.PlayerDetailResponse{}, nil
}

type stubSyncRunService struct{}

func (stubSyncRunService) Trigger(ctx context.Context, typ model.SyncType, actorID *int64) (dto.SyncRunResponse, error) {
	return dto.SyncRunResponse{}, nil
}

func (stubSyncRunService) GetRun(ctx context.Context, id int64) (dto.SyncRunResponse, error) {
	return dto.SyncRunResponse{}, nil
}

func (stubSyncRunService) ListRuns(ctx context.Context, typ *model.SyncType, offset, limit int) (dto.SyncRunPageResponse, error) {
	return dto.SyncRunPageResponse{}, nil
}

func (stubSyncRunService) ListSchedules(ctx context.Context) ([]dto.SyncScheduleResponse, error) {
	return nil, nil
}

func (stubSyncRunService) UpdateSchedule(ctx context.Context, typ model.SyncType, enabled bool, intervalMinutes int, actorID int64) (dto.SyncScheduleResponse, error) {
	return dto.SyncScheduleResponse{}, nil
}

type stubPlayerStatisticsService struct{}

func (stubPlayerStatisticsService) GetPlayerStatistics(ctx context.Context, playerID int64) ([]dto.PlayerStatisticsResponse, error) {
	return nil, nil
}

type stubAuthService struct{}

func (stubAuthService) Register(ctx context.Context, email, password string) (model.User, error) {
	return model.User{}, nil
}

func (stubAuthService) Login(ctx context.Context, email, password string) (model.Session, error) {
	return model.Session{}, nil
}

func (stubAuthService) Refresh(ctx context.Context, presentedID string, userID int64, sessionID string) (model.Session, error) {
	return model.Session{}, nil
}

func (stubAuthService) Logout(ctx context.Context, userID int64, sessionID string) error {
	return nil
}

type stubScoringService struct{}

func (stubScoringService) ListStrategies() []dto.ScoringStrategyResponse { return nil }

func (stubScoringService) ActiveStrategy(ctx context.Context) (dto.ActiveStrategyResponse, bool, error) {
	return dto.ActiveStrategyResponse{}, false, nil
}

func (stubScoringService) ActivateStrategy(ctx context.Context, strategyID int, actorID int64) error {
	return nil
}

func (stubScoringService) SetLeagueCoefficient(ctx context.Context, leagueCode string, coefficient float64, actorID int64) error {
	return nil
}

func (stubScoringService) GetLeagueCoefficient(ctx context.Context, leagueCode string) (dto.LeagueCoefficientResponse, bool, error) {
	return dto.LeagueCoefficientResponse{}, false, nil
}

func (stubScoringService) GetScore(ctx context.Context, playerID int64) (dto.PlayerScoreResponse, error) {
	return dto.PlayerScoreResponse{}, nil
}

func (stubScoringService) GetScoreHistory(ctx context.Context, playerID int64) ([]dto.PlayerScoreResponse, error) {
	return nil, nil
}

func (stubScoringService) ListRanking(ctx context.Context, filter dto.PlayerFilterDTO, sortOrder string, offset int) (dto.PlayerRankingPageResponse, error) {
	return dto.PlayerRankingPageResponse{}, nil
}

func (s stubController) PlaceBuyOrder() httphandler.Endpoint  { return s.endpoint() }
func (s stubController) PlaceSellOrder() httphandler.Endpoint { return s.endpoint() }
func (s stubController) ListOpenOrders() httphandler.Endpoint { return s.endpoint() }
func (s stubController) CancelOrder() httphandler.Endpoint    { return s.endpoint() }
func (s stubController) Portfolio() httphandler.Endpoint      { return s.endpoint() }
func (s stubController) Transactions() httphandler.Endpoint   { return s.endpoint() }

func routesUnderTest() []route {
	server := &Server{
		controllers: &controller.Container{
			Player: controller.NewPlayerController(stubPlayerService{}),
			Sync: controller.NewSyncController(
				stubPlayerStatisticsService{},
				stubSyncRunService{},
			),
			Scoring: controller.NewScoringController(stubScoringService{}),
			Auth:    controller.NewAuthController(stubAuthService{}),
			Market:  controller.NewMarketController(nil, nil),
		},
	}

	return server.routes()
}

func TestEveryRouteDeclaresAnAccessLevel(t *testing.T) {
	for _, r := range routesUnderTest() {
		if r.access == AccessUndeclared {
			t.Errorf("la ruta %q no declara su nivel de acceso", r.pattern)
		}
		if r.endpoint == nil {
			t.Errorf("la ruta %q no tiene endpoint", r.pattern)
		}
		if r.pattern == "" {
			t.Error("hay una ruta sin patrón")
		}
	}
}

func TestDeclaredLevelsMatchTheSpecification(t *testing.T) {
	expected := map[string]AccessLevel{
		"POST /auth/register":                       AccessAnonymous,
		"POST /auth/login":                          AccessAnonymous,
		"POST /auth/refresh":                        AccessRenewal,
		"POST /auth/logout":                         AccessAuthenticated,
		"GET /players":                              AccessAnonymous,
		"GET /players/{id}":                         AccessAnonymous,
		"GET /players/{id}/statistics":              AccessAnonymous,
		"POST /sync/{type}/runs":                    AccessSuperuser,
		"GET /sync/runs":                            AccessSuperuser,
		"GET /sync/runs/{id}":                       AccessSuperuser,
		"GET /sync/schedules":                       AccessSuperuser,
		"PUT /sync/schedules/{type}":                AccessSuperuser,
		"GET /scoring/strategies":                   AccessAnonymous,
		"GET /scoring/active-strategy":              AccessAnonymous,
		"PUT /scoring/active-strategy":              AccessSuperuser,
		"GET /scoring/league-coefficients/{league}": AccessAnonymous,
		"PUT /scoring/league-coefficients/{league}": AccessSuperuser,
		"GET /players/{id}/score":                   AccessAnonymous,
		"GET /players/{id}/scores":                  AccessAnonymous,
		"GET /players/ranking":                      AccessAnonymous,
		"POST /orders/buy":                          AccessAuthenticated,
		"POST /orders/sell":                         AccessAuthenticated,
		"GET /orders":                               AccessAuthenticated,
		"DELETE /orders/{id}":                       AccessAuthenticated,
		"GET /users/{id}/portfolio":                 AccessAuthenticated,
		"GET /users/{id}/transactions":              AccessAuthenticated,
	}

	declared := map[string]AccessLevel{}
	for _, r := range routesUnderTest() {
		if _, duplicated := declared[r.pattern]; duplicated {
			t.Errorf("la ruta %q está declarada dos veces", r.pattern)
		}
		declared[r.pattern] = r.access
	}

	for pattern, want := range expected {
		got, found := declared[pattern]
		if !found {
			t.Errorf("falta la ruta %q, que la especificación exige", pattern)
			continue
		}
		if got != want {
			t.Errorf("la ruta %q está declarada %s y la especificación la fija en %s", pattern, got, want)
		}
	}

	for pattern, level := range declared {
		if _, expectedRoute := expected[pattern]; !expectedRoute {
			t.Errorf("la ruta %q está declarada como %s y no figura en la especificación: "+
				"agregarla al mapa de este caso es la decisión de dejarla expuesta", pattern, level)
		}
	}
}

// La declaración y lo que se aplica tienen que coincidir: un nivel que exige
// credencial no puede terminar sirviéndose sin la cadena que la pide. buildMux
// entra en pánico ante un nivel que no sabe atender, y eso pasa al arrancar y
// no en la primera petición.
func TestBuildMuxPanicsOnAnUndeclaredLevel(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("buildMux registró una ruta sin nivel declarado en vez de entrar en pánico")
		}
		if message, ok := recovered.(string); !ok || !strings.Contains(message, "/sin-declarar") {
			t.Errorf("el pánico no nombra la ruta culpable: %v", recovered)
		}
	}()

	undeclared := []route{{pattern: "GET /sin-declarar", endpoint: stubController{}.endpoint()}}

	buildMux(undeclared, nil, nil)
}

// stubVerifier acepta toda credencial y declara el privilegio que se le pida.
type stubVerifier struct {
	privilege model.PrivilegeLevel
}

func (s stubVerifier) Verify(string) (adapters.Claims, error) {
	return adapters.Claims{Subject: 42, Privilege: s.privilege, Kind: adapters.KindAccess}, nil
}

// El orden de la cadena de AccessSuperuser, ejercitado a través de buildMux y
// no armado a mano.
//
// Existe porque la composición se lee al revés de como se ejecuta y es fácil
// invertirla: con la autorización por fuera, corre antes de que la
// autenticación haya publicado el actor, y toda operación de superusuario
// termina en 401 sin que nadie se entere de por qué. Los casos del middleware
// no lo detectan porque arman su propia cadena; esto mira la que se usa.
func TestSuperuserChainAuthenticatesBeforeAuthorizing(t *testing.T) {
	cases := []struct {
		name      string
		privilege model.PrivilegeLevel
		want      int
	}{
		{"el superusuario pasa", model.PrivilegeSuperuser, http.StatusOK},
		{"un usuario comun recibe 403 y no 401", model.PrivilegeUser, http.StatusForbidden},
		{"un privilegio desconocido recibe 403", model.PrivilegeUnknown, http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reached := false
			routes := []route{{
				pattern: "GET /solo-superusuario",
				access:  AccessSuperuser,
				endpoint: func(w http.ResponseWriter, _ *http.Request) error {
					reached = true
					return httphandler.Encode(w, http.StatusOK, map[string]string{"status": "ok"})
				},
			}}

			mux := buildMux(
				routes,
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				middleware.NewContainer(stubVerifier{privilege: c.privilege}),
			)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/solo-superusuario", nil)
			request.Header.Set("Authorization", "Bearer una-credencial")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != c.want {
				t.Errorf("status = %d, se esperaba %d: %s", recorder.Code, c.want, recorder.Body.String())
			}
			if reached != (c.want == http.StatusOK) {
				t.Errorf("la operacion %s corrio", map[bool]string{true: "si", false: "no"}[reached])
			}
		})
	}
}

// Y sin credencial la misma cadena da 401, que es el otro lado de el requerimiento.
func TestSuperuserChainRefusesWithoutACredential(t *testing.T) {
	routes := []route{{
		pattern:  "GET /solo-superusuario",
		access:   AccessSuperuser,
		endpoint: stubController{}.endpoint(),
	}}

	mux := buildMux(
		routes,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		middleware.NewContainer(stubVerifier{privilege: model.PrivilegeSuperuser}),
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/solo-superusuario", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba 401", recorder.Code)
	}
}
