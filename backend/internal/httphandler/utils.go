package httphandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Validator lo implementa el DTO que valida la forma de sus propios campos.
// Decode lo invoca cuando está presente, así que ningún endpoint puede
// saltearse la validación por olvidarse de llamarla.
type Validator interface {
	Validate() error
}

// Encode sets the content type, status code, and JSON-encodes the given data into the response.
func Encode[T any](w http.ResponseWriter, status int, data T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// Decode reads the JSON from the request body and unmarshals it into the given type.
//
// Es el único lugar por el que pasa todo cuerpo, así que es donde viven los
// tres chequeos que todo endpoint debe hacer: campos inesperados rechazados en
// vez de ignorados, un único valor JSON por cuerpo, y la validación propia del
// DTO. Cualquiera de los tres falla con 400 antes de que corra lógica de
// negocio o se toque la persistencia.
func Decode[T any](r *http.Request) (T, error) {
	var v T

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&v); err != nil {
		return v, NewError(http.StatusBadRequest, badRequestMessage(err), err)
	}

	// Un segundo valor JSON detrás del primero significa que el cuerpo no es
	// lo que dice ser. json.Decoder lee de a un valor y se quedaría con el
	// primero sin decir nada.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		return v, NewError(http.StatusBadRequest, "el cuerpo contiene más de un valor JSON", err)
	}

	if validator, ok := any(&v).(Validator); ok {
		if err := validator.Validate(); err != nil {
			return v, NewError(http.StatusBadRequest, err.Error(), err)
		}
	}

	return v, nil
}

// badRequestMessage traduce el error de json a algo que le sirva a quien hizo
// la petición. Los mensajes de encoding/json nombran el campo que molestó y no
// devuelven los valores que se mandaron, que es justo lo que hace falta: decir
// qué estuvo mal sin repetir un secreto.
func badRequestMessage(err error) string {
	var unmarshalTypeError *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeError) {
		return fmt.Sprintf("el campo %q no tiene el tipo esperado", unmarshalTypeError.Field)
	}

	if errors.Is(err, io.EOF) {
		return "el cuerpo de la petición está vacío"
	}

	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return "el cuerpo de la petición no es JSON válido"
	}

	// DisallowUnknownFields no expone un tipo propio: su error es un errors.errorString
	// cuyo texto ya nombra el campo sobrante, que es exactamente lo que se quiere
	// informar. Se le saca el prefijo "json: " para que el mensaje sea el que
	// documenta el contrato y no uno que delate la librería que lo produjo.
	return strings.TrimPrefix(err.Error(), "json: ")
}
