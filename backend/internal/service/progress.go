package service

import "context"

// Progress es el avance efímero de una sincronización. Viaja por contexto y no
// se persiste: es un informe para quien está mirando la ejecución.
type Progress struct {
	Message string
	Current int
	Total   int
}

// Reporter recibe actualizaciones de progreso. Es un callback, no una interfaz,
// para que cada emisor no tenga que declarar un tipo propio.
type Reporter func(Progress)

type progressContextKey struct{}

// WithProgress publica un reporter en el contexto. Los services de sync lo
// consultan y, si está presente, emiten un avance por liga; si no, no hacen
// nada, así que su firma pública no cambia.
func WithProgress(ctx context.Context, reporter Reporter) context.Context {
	return context.WithValue(ctx, progressContextKey{}, reporter)
}

// FromContext devuelve el reporter publicado, si lo hay.
func FromContext(ctx context.Context) (Reporter, bool) {
	reporter, ok := ctx.Value(progressContextKey{}).(Reporter)
	return reporter, ok
}

// reportProgress emite un avance si el contexto lleva reporter.
func reportProgress(ctx context.Context, current, total int, message string) {
	if reporter, ok := FromContext(ctx); ok {
		reporter(Progress{Current: current, Total: total, Message: message})
	}
}
