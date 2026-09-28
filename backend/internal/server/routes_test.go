// Este archivo es el chequeo automático que pide FR-014, en dos partes: una
// tabla sobre lo que routes() declara, y una lectura del código fuente que
// detecta una ruta registrada por fuera de buildMux.
//
// Vive en el paquete server y no en server_test porque necesita llamar a
// routes(), que no está exportado. Los dos paquetes de test conviven en el
// directorio.
package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
)

// Los controllers de estos casos no hacen nada: lo que se examina es la tabla
// de rutas, no lo que cada endpoint responde.
type stubController struct{}

func (stubController) endpoint() httphandler.Endpoint {
	return func(http.ResponseWriter, *http.Request) error { return nil }
}

func (s stubController) GetPlayers() httphandler.Endpoint { return s.endpoint() }
func (s stubController) Register() httphandler.Endpoint   { return s.endpoint() }
func (s stubController) Login() httphandler.Endpoint      { return s.endpoint() }

func routesUnderTest() []route {
	server := &Server{
		controllers: &controller.Container{
			Player: stubController{},
			Auth:   stubController{},
		},
	}

	return server.routes()
}

// La mitad que FR-014 pide literalmente: ninguna ruta puede quedarse sin nivel.
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

// FR-015 fija el nivel de cada endpoint de esta entrega. El mapa se compara
// exacto en las dos direcciones, así que una ruta nueva que no se agregue acá
// también hace fallar el caso.
func TestDeclaredLevelsMatchTheSpecification(t *testing.T) {
	expected := map[string]AccessLevel{
		"POST /auth/register": AccessAnonymous,
		"POST /auth/login":    AccessAnonymous,
		"GET /players":        AccessAuthenticated,
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

// La otra mitad, la que FR-014 no nombra pero que es el mismo agujero con otra
// causa: un endpoint que nunca entró en la tabla.
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
