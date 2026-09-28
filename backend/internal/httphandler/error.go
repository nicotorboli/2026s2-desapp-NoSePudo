package httphandler

import "fmt"

// Error es la implementación concreta de APIError. Lleva el status que el
// cliente va a recibir, el mensaje que se le muestra y la causa original
// envuelta, que queda disponible para el log y para errors.Is/errors.As pero
// nunca se le responde al cliente.
//
// La separación importa: Message() es vocabulario de borde y no debe filtrar
// nada de lo que el cliente mandó (FR-025), mientras que la causa es lo que
// hace diagnosticable el error tres capas más abajo.
type Error struct {
	cause   error
	message string
	status  int
}

// NewError construye un error de borde. cause puede ser nil cuando la
// refutación nace acá mismo, que es el caso del middleware.
func NewError(status int, message string, cause error) *Error {
	return &Error{
		cause:   cause,
		message: message,
		status:  status,
	}
}

func (e *Error) Error() string {
	if e.cause == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

func (e *Error) StatusCode() int {
	return e.status
}

func (e *Error) Message() string {
	return e.message
}

// Unwrap expone la causa para errors.Is y errors.As, y es lo que permite que
// un centinela de dominio siga siendo reconocible después de haber sido
// traducido a un error HTTP.
func (e *Error) Unwrap() error {
	return e.cause
}
