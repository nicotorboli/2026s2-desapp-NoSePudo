package httphandler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
)

type plainBody struct {
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type validatedBody struct {
	Email string `json:"email"`
}

var errEmailRequired = errors.New("el campo email es obligatorio")

func (b *validatedBody) Validate() error {
	if b.Email == "" {
		return errEmailRequired
	}
	return nil
}

// countingBody cuenta las invocaciones para probar que Decode llama a Validate
// exactamente una vez.
type countingBody struct {
	Name  string
	Calls int `json:"-"`
}

func (b *countingBody) Validate() error {
	b.Calls++
	return nil
}

func requestWith(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
}

// assertBadRequest comprueba que el error que devolvió Decode llega al cliente
// como un 400 y no como un error interno.
func assertBadRequest(t *testing.T, err error) {
	t.Helper()
	badRequestMessage(t, err)
}

// badRequestMessage hace el mismo chequeo y además devuelve el mensaje que se
// le responde al cliente, para los casos que afirman qué campo se nombró.
func badRequestMessage(t *testing.T, err error) string {
	t.Helper()

	if err == nil {
		t.Fatal("se esperaba un error y Decode no devolvió ninguno")
	}

	apiErr, ok := errors.AsType[*httphandler.Error](err)
	if !ok {
		t.Fatalf("el error no es un httphandler.Error, así que el wrapper lo respondería como 500: %v", err)
	}
	if apiErr.StatusCode() != http.StatusBadRequest {
		t.Errorf("status = %d, se esperaba %d", apiErr.StatusCode(), http.StatusBadRequest)
	}

	return apiErr.Message()
}

func TestDecodeAcceptsAWellFormedBody(t *testing.T) {
	got, err := httphandler.Decode[plainBody](requestWith(t, `{"email":"nico@nosepudo.ar","age":30}`))
	if err != nil {
		t.Fatalf("Decode devolvió error: %v", err)
	}

	if got.Email != "nico@nosepudo.ar" || got.Age != 30 {
		t.Errorf("Decode = %+v, se esperaba el cuerpo decodificado", got)
	}
}

// el requerimiento: un campo inesperado se rechaza, no se ignora en silencio. Es también
// lo que hace imposible que un privilege del cliente llegue a una cuenta.
func TestDecodeRejectsAnUnknownField(t *testing.T) {
	_, err := httphandler.Decode[plainBody](requestWith(t, `{"email":"nico@nosepudo.ar","privilege":"superuser"}`))

	message := badRequestMessage(t, err)
	if !strings.Contains(message, "privilege") {
		t.Errorf("el mensaje %q no nombra el campo sobrante", message)
	}
}

func TestDecodeRejectsContentAfterTheFirstJSONValue(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"un segundo objeto", `{"email":"a@b.c"}{"email":"d@e.f"}`},
		{"un valor suelto detrás", `{"email":"a@b.c"} 42`},
		{"basura detrás", `{"email":"a@b.c"} no-json`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := httphandler.Decode[plainBody](requestWith(t, c.body))
			assertBadRequest(t, err)
		})
	}
}

func TestDecodeAllowsTrailingWhitespace(t *testing.T) {
	if _, err := httphandler.Decode[plainBody](requestWith(t, "{\"email\":\"a@b.c\"}\n \t\n")); err != nil {
		t.Errorf("un salto de línea final no debería ser un error: %v", err)
	}
}

func TestDecodeRejectsAnEmptyBody(t *testing.T) {
	_, err := httphandler.Decode[plainBody](requestWith(t, ""))
	assertBadRequest(t, err)
}

func TestDecodeRejectsMalformedJSON(t *testing.T) {
	_, err := httphandler.Decode[plainBody](requestWith(t, `{"email":`))
	assertBadRequest(t, err)
}

func TestDecodeRejectsAFieldOfTheWrongType(t *testing.T) {
	_, err := httphandler.Decode[plainBody](requestWith(t, `{"email":"a@b.c","age":"treinta"}`))

	message := badRequestMessage(t, err)
	if !strings.Contains(message, "age") {
		t.Errorf("el mensaje %q no nombra el campo con el tipo equivocado", message)
	}
}

// R10: el DTO que define Validate siempre se valida, sin que el controller
// tenga que acordarse de llamarla.
func TestDecodeRunsValidateWhenTheTypeImplementsValidator(t *testing.T) {
	_, err := httphandler.Decode[validatedBody](requestWith(t, `{"email":""}`))

	message := badRequestMessage(t, err)
	if message != errEmailRequired.Error() {
		t.Errorf("mensaje = %q, se esperaba el del DTO: %q", message, errEmailRequired)
	}
	if !errors.Is(err, errEmailRequired) {
		t.Error("el error de validación original no es recuperable con errors.Is")
	}
}

func TestDecodeReturnsTheValueWhenValidatePasses(t *testing.T) {
	got, err := httphandler.Decode[validatedBody](requestWith(t, `{"email":"nico@nosepudo.ar"}`))
	if err != nil {
		t.Fatalf("Decode devolvió error: %v", err)
	}
	if got.Email != "nico@nosepudo.ar" {
		t.Errorf("Decode = %+v", got)
	}
}

func TestDecodeCallsValidateExactlyOnce(t *testing.T) {
	got, err := httphandler.Decode[countingBody](requestWith(t, `{"Name":"nico"}`))
	if err != nil {
		t.Fatalf("Decode devolvió error: %v", err)
	}
	if got.Calls != 1 {
		t.Errorf("Validate se llamó %d veces, se esperaba 1", got.Calls)
	}
}

func TestDecodeLeavesATypeWithoutValidatorAlone(t *testing.T) {
	got, err := httphandler.Decode[plainBody](requestWith(t, `{"email":"","age":0}`))
	if err != nil {
		t.Fatalf("un tipo sin Validate no debería pagar nada: %v", err)
	}
	if got.Email != "" || got.Age != 0 {
		t.Errorf("Decode = %+v, se esperaban los valores cero", got)
	}
}
