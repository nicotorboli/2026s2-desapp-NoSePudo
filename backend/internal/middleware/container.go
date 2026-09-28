package middleware

// Container arma los middlewares de la capa. Los recibe quien construye el
// router, que es el único que decide qué cadena le corresponde a cada ruta.
type Container struct {
	Authentication *Authentication
	Authorization  *Authorization
}

func NewContainer(tokenVerifier TokenVerifier) *Container {
	return &Container{
		Authentication: NewAuthentication(tokenVerifier),
		Authorization:  NewAuthorization(),
	}
}
