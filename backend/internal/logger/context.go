package logger

import (
	"context"
	"log/slog"
)

// contextKey es un tipo propio y no exportado, así que ninguna otra clave
// puede colisionar con la nuestra ni leer lo que guardamos.
type contextKey struct{}

// Into deja el logger en el contexto para que las capas de abajo lo tomen con
// FromContext. Lo usa el middleware de autenticación cuando le agrega el actor.
func Into(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

// FromContext devuelve el logger que viaja en el contexto, o el logger por
// defecto cuando todavía no hay ninguno.
//
// Es el contrato que esta feature necesita de la feature de observabilidad
// todavía no especificada: cuando exista, lo que devuelva va a venir ya
// decorado con el identificador de correlación, y el actor que agrega el
// middleware va a quedar al lado sin que nadie tenga que enhebrar nada por las
// firmas. Hasta entonces esto es correcto pero poco informativo.
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}
