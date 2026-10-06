// Package docs sirve la documentación interactiva de la API (Swagger UI) y el
// contrato que la alimenta. Vive deliberadamente fuera de routes()/buildMux:
// no es un endpoint de negocio, no declara nivel de acceso, y por eso
// routes_test.go y contract_test.go no lo ven — ver research.md §4 de
// specs/004-documentacion-swagger.
package docs

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
)

// docsPrefix es el subárbol donde se sirve el shell de Swagger UI embebido.
const docsPrefix = "/docs/"

// openAPIPath es la ruta exacta por la que se sirve el contrato crudo.
const openAPIPath = "/openapi.yaml"

// dist contiene los assets de Swagger UI vendorizados, embebidos en el
// binario para que la página funcione sin acceso a internet y sin depender
// del directorio de trabajo del proceso (FR-004).
//
//go:embed dist
var dist embed.FS

// assetServer sirve dist/ (embebido) como si "dist" fuera la raíz, para que
// "/docs/swagger-ui.css" resuelva a "dist/swagger-ui.css" sin que el llamador
// tenga que saber del prefijo del embed.
var assetServer = func() http.Handler {
	assets, err := fs.Sub(dist, "dist")
	if err != nil {
		// dist está embebido en tiempo de compilación: si "dist" no existe
		// adentro, el binario está mal construido y no hay petición que lo
		// arregle.
		panic("docs: no se pudo abrir el subárbol embebido dist: " + err.Error())
	}
	return http.FileServer(http.FS(assets))
}()

// Mount antepone la documentación a next: "/openapi.yaml" (match exacto) y
// cualquier ruta bajo "/docs/" (match de prefijo) se sirven acá; todo lo
// demás cae a next sin tocarse, per contracts/docs-endpoints.md.
func Mount(contractPath string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == openAPIPath:
			serveContract(contractPath, w, r)
		case len(r.URL.Path) >= len(docsPrefix) && r.URL.Path[:len(docsPrefix)] == docsPrefix:
			http.StripPrefix(docsPrefix, assetServer).ServeHTTP(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

// serveContract lee contractPath del disco en cada petición (FR-003: un
// cambio en el archivo se ve en el próximo request, sin reiniciar) y lo
// devuelve byte a byte. Un error de lectura se loguea y responde 500 en vez
// de hacer panic, para que el resto del backend siga sirviendo (FR-012).
func serveContract(contractPath string, w http.ResponseWriter, r *http.Request) {
	raw, err := os.ReadFile(contractPath) //nolint:gosec // ruta fija, inyectada por quien construye el servidor
	if err != nil {
		logger.FromContext(r.Context()).ErrorContext(r.Context(), "no se pudo leer el contrato OpenAPI",
			"path", contractPath, "error", err.Error())
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("openapi contract unavailable"))
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
