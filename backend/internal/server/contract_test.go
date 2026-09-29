package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// contractPath es el documento mantenido que exige el Principio VII, no la copia
// que quedó en el directorio de la especificación. Esa es el registro de lo que
// se diseñó para una feature; este describe la API que hay.
var contractPath = filepath.Join("..", "..", "api", "openapi.yaml")

// contract es la parte del documento que a estos casos les interesa: qué rutas
// declara y con qué nivel de acceso.
type contract struct {
	Paths map[string]map[string]struct {
		AccessLevel string `yaml:"x-access-level"`
		OperationID string `yaml:"operationId"`
	} `yaml:"paths"`
}

func loadContract(t *testing.T) contract {
	t.Helper()

	// La ruta es una constante del paquete, no algo que llegue de afuera.
	raw, err := os.ReadFile(contractPath) //nolint:gosec // ruta fija del repositorio
	if err != nil {
		t.Fatalf("no se pudo leer el contrato publicado: %v", err)
	}

	var document contract
	if err := yaml.Unmarshal(raw, &document); err != nil {
		t.Fatalf("el contrato no es YAML válido: %v", err)
	}
	if len(document.Paths) == 0 {
		t.Fatal("el contrato no declara ninguna ruta")
	}

	return document
}

// documentedRoutes traduce el documento a la misma forma que la tabla de rutas:
// "MÉTODO /ruta" contra nivel de acceso.
func documentedRoutes(t *testing.T, document contract) map[string]string {
	t.Helper()

	documented := map[string]string{}

	for path, operations := range document.Paths {
		for method, operation := range operations {
			pattern := strings.ToUpper(method) + " " + path

			if operation.AccessLevel == "" {
				t.Errorf("la operación %q no declara x-access-level en el contrato", pattern)
				continue
			}
			documented[pattern] = operation.AccessLevel
		}
	}

	return documented
}

// El documento y la tabla de rutas tienen que decir lo mismo, en las dos
// direcciones.
//
// Es lo que mantiene honesto al contrato: un endpoint que se agregue sin
// documentar rompe un caso, y uno que se documente con un nivel que el código no
// aplica también. Sin esto el documento envejece en silencio, que es
// exactamente lo que el Principio VII quiere evitar.
func TestPublishedContractMatchesTheRouteTable(t *testing.T) {
	documented := documentedRoutes(t, loadContract(t))

	declared := map[string]string{}
	for _, r := range routesUnderTest() {
		declared[r.pattern] = r.access.String()
	}

	for pattern, level := range declared {
		documentedLevel, found := documented[pattern]
		if !found {
			t.Errorf("la ruta %q está registrada y no figura en el contrato publicado", pattern)
			continue
		}
		if documentedLevel != level {
			t.Errorf("la ruta %q se aplica como %s y el contrato la documenta como %s",
				pattern, level, documentedLevel)
		}
	}

	for pattern := range documented {
		if _, registered := declared[pattern]; !registered {
			t.Errorf("el contrato documenta la ruta %q, que no está registrada", pattern)
		}
	}
}

// Los niveles que el documento usa tienen que ser los cuatro que existen. Un
// nivel inventado en el contrato sería una descripción que el código no puede
// aplicar.
func TestContractUsesOnlyKnownAccessLevels(t *testing.T) {
	known := map[string]bool{
		AccessAnonymous.String(): true,
		AccessAuthenticated.String(): true,
		AccessRenewal.String(): true,
		AccessSuperuser.String(): true,
	}

	for pattern, level := range documentedRoutes(t, loadContract(t)) {
		if !known[level] {
			t.Errorf("la ruta %q declara el nivel %q, que no es uno de los cuatro", pattern, level)
		}
	}
}

// Toda operación lleva operationId, que es lo que un generador de clientes usa
// para nombrar el método.
func TestContractOperationsAreIdentified(t *testing.T) {
	document := loadContract(t)

	seen := map[string]string{}
	for path, operations := range document.Paths {
		for method, operation := range operations {
			pattern := strings.ToUpper(method) + " " + path

			if operation.OperationID == "" {
				t.Errorf("la operación %q no tiene operationId", pattern)
				continue
			}
			if previous, duplicated := seen[operation.OperationID]; duplicated {
				t.Errorf("el operationId %q está en %q y en %q", operation.OperationID, previous, pattern)
			}
			seen[operation.OperationID] = pattern
		}
	}
}

