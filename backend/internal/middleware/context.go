package middleware

import "context"

// actorContextKey es un tipo propio y no exportado, así que ningún otro
// paquete puede escribir ni leer esta entrada del contexto por accidente.
type actorContextKey struct{}

// WithActor publica el actor para las capas de abajo. Lo llama únicamente el
// middleware de autenticación, y recién después de haber verificado la
// credencial.
func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

// ActorFromContext devuelve el actor de la petición. El segundo valor es
// falso cuando la petición no pasó por autenticación, que es lo que distingue
// "anónimo" de "autenticado como la cuenta cero".
func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	return actor, ok
}
