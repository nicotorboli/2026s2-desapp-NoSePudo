package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// stubNext representa el router de negocio: si Mount lo invoca, es porque la
// petición no era de documentación.
func stubNext() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
}

func TestMountServesTheEmbeddedDocumentationShell(t *testing.T) {
	cases := []struct {
		name            string
		path            string
		wantContentType string
		wantStatus      int
	}{
		{"la raíz de /docs/ sirve index.html", "/docs/", "text/html; charset=utf-8", http.StatusOK},
		{"un asset embebido se sirve", "/docs/swagger-ui.css", "text/css; charset=utf-8", http.StatusOK},
		{"un asset que no existe da 404", "/docs/does-not-exist", "", http.StatusNotFound},
	}

	mux := Mount(filepath.Join("..", "..", "..", "api", "openapi.yaml"), stubNext())

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, c.path, nil)
			mux.ServeHTTP(recorder, request)

			if recorder.Code != c.wantStatus {
				t.Errorf("status = %d, se esperaba %d", recorder.Code, c.wantStatus)
			}
			if c.wantContentType != "" && recorder.Header().Get("Content-Type") != c.wantContentType {
				t.Errorf("Content-Type = %q, se esperaba %q", recorder.Header().Get("Content-Type"), c.wantContentType)
			}
		})
	}
}

func TestMountFallsThroughToNextForEverythingElse(t *testing.T) {
	mux := Mount(filepath.Join("..", "..", "..", "api", "openapi.yaml"), stubNext())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/players", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTeapot {
		t.Errorf("status = %d, se esperaba %d (que la petición haya caído a next)", recorder.Code, http.StatusTeapot)
	}
}

func TestOpenAPIYamlServesTheFileFromDiskByteForByte(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "openapi.yaml")
	want := []byte("openapi: 3.0.0\ninfo:\n  title: fixture\n")
	if err := os.WriteFile(fixture, want, 0o600); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	mux := Mount(fixture, stubNext())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.yaml", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/yaml" {
		t.Errorf("Content-Type = %q, se esperaba %q", got, "application/yaml")
	}
	if recorder.Body.String() != string(want) {
		t.Errorf("body = %q, se esperaba %q (byte a byte)", recorder.Body.String(), string(want))
	}
}

func TestOpenAPIYamlRespondsWithA500WhenTheFileIsMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-existe.yaml")

	mux := Mount(missing, stubNext())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.yaml", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, se esperaba 500", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, se esperaba %q", got, "text/plain; charset=utf-8")
	}
}

// Regresión para US3 (FR-003): editar el archivo en disco tiene que
// reflejarse en la próxima petición, sin reinicio ni caché de ningún tipo.
func TestOpenAPIYamlReflectsAnEditWithNoRestart(t *testing.T) {
	fixture := filepath.Join(t.TempDir(), "openapi.yaml")
	original := []byte("openapi: 3.0.0\ninfo:\n  title: original\n")
	if err := os.WriteFile(fixture, original, 0o600); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	mux := Mount(fixture, stubNext())

	first := httptest.NewRecorder()
	mux.ServeHTTP(first, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.yaml", nil))
	if first.Body.String() != string(original) {
		t.Fatalf("primera lectura = %q, se esperaba el contenido original", first.Body.String())
	}

	updated := []byte("openapi: 3.0.0\ninfo:\n  title: actualizado\n")
	if err := os.WriteFile(fixture, updated, 0o600); err != nil {
		t.Fatalf("no se pudo sobreescribir el fixture: %v", err)
	}

	second := httptest.NewRecorder()
	mux.ServeHTTP(second, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/openapi.yaml", nil))
	if second.Body.String() != string(updated) {
		t.Errorf("segunda lectura = %q, se esperaba el contenido actualizado sin reinicio", second.Body.String())
	}
}
