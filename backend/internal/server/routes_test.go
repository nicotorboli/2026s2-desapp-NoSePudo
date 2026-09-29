// Verificaciones sobre la tabla de rutas routes() y que no existan
// rutas registradas por fuera de buildMux.
//
// Vive en el paquete server porque necesita llamar a routes(), que no está exportado.
package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// Los controllers de estos casos no hacen nada: lo que se examina es la tabla
// de rutas, no lo que cada endpoint responde.
type stubController struct{}

func (stubController) endpoint() httphandler.Endpoint {
	return func(http.ResponseWriter, *http.Request) error { return nil }
}

func (s stubController) GetPlayers() httphandler.Endpoint    { return s.endpoint() }
func (s stubController) GetPlayerByID() httphandler.Endpoint { return s.endpoint() }
func (s stubController) SyncPlayers() httphandler.Endpoint   { return s.endpoint() }
func (s stubController) Register() httphandler.Endpoint      { return s.endpoint() }
func (s stubController) Login() httphandler.Endpoint         { return s.endpoint() }
func (s stubController) Refresh() httphandler.Endpoint       { return s.endpoint() }
func (s stubController) Logout() httphandler.Endpoint        { return s.endpoint() }

func routesUnderTest() []route {
	server := &Server{
		controllers: &controller.Container{
			Player: stubController{},
			Sync:   stubController{},
			Auth:   stubController{},
		},
	}

	return server.routes()
}

// Ninguna ruta puede quedarse sin nivel de acceso declarado.
// Un literal de route que se olvide del campo obtiene AccessUndeclared, y acá
// se entera.
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

// Comprueba el nivel esperado de cada endpoint declarado. El mapa se compara
// exacto en las dos direcciones, así que una ruta nueva que no se agregue acá
// también hace fallar el caso.
func TestDeclaredLevelsMatchTheSpecification(t *testing.T) {
	expected := map[string]AccessLevel{
		"POST /auth/register": AccessAnonymous,
		"POST /auth/login":    AccessAnonymous,
		"POST /auth/refresh":  AccessRenewal,
		"POST /auth/logout":   AccessAuthenticated,
		"GET /players":        AccessAuthenticated,
		"GET /players/{id}":   AccessAuthenticated,
		"POST /players/sync":  AccessSuperuser,
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

// Garantiza que ningún endpoint se registre por fuera de buildMux.
//
// Go no ofrece manera de enumerar los patrones registrados en un ServeMux, así
// que el código fuente es la única otra fuente de verdad. Se lee con go/ast y
// no con un grep porque un grep se rompe con un comentario o una cadena que
// contenga "router.Handle".
func TestNoRouteIsRegisteredOutsideBuildMux(t *testing.T) {
	const registrar = "buildMux"

	fileSet := token.NewFileSet()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("no se pudo leer el directorio del paquete: %v", err)
	}

	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++

		file, err := parser.ParseFile(fileSet, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("no se pudo parsear %s: %v", name, err)
		}

		for _, declaration := range file.Decls {
			function, isFunction := declaration.(*ast.FuncDecl)
			if !isFunction || function.Name.Name == registrar {
				continue
			}

			ast.Inspect(function, func(node ast.Node) bool {
				call, isCall := node.(*ast.CallExpr)
				if !isCall {
					return true
				}
				selector, isSelector := call.Fun.(*ast.SelectorExpr)
				if !isSelector {
					return true
				}

				if selector.Sel.Name == "Handle" || selector.Sel.Name == "HandleFunc" {
					position := fileSet.Position(selector.Sel.Pos())
					t.Errorf(
						"%s:%d: %s llama a %s por fuera de %s: la ruta no pasa por la tabla "+
							"y queda sin el nivel de acceso que le corresponde",
						name, position.Line, function.Name.Name, selector.Sel.Name, registrar,
					)
				}

				return true
			})
		}
	}

	// Si el filtro dejara de encontrar archivos, el caso pasaría sin haber
	// mirado nada.
	if checked == 0 {
		t.Fatal("no se examinó ningún archivo del paquete")
	}
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
