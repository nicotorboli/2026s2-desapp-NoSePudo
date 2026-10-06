window.onload = function() {
  window.ui = SwaggerUIBundle({
    url: "/openapi.yaml",
    dom_id: '#swagger-ui',
    deepLinking: true,
    persistAuthorization: true,
    // BaseLayout (el default): sin la barra "Explore" de StandaloneLayout, que
    // sirve para apuntar a un spec a elección — acá el spec es fijo, así que
    // esa barra no tiene nada que hacer.
    presets: [
      SwaggerUIBundle.presets.apis
    ],
    // Sin el validador en línea de swagger.io: es el único pedido saliente que
    // Swagger UI hace por su cuenta, y esta página tiene que andar sin
    // acceso a internet (FR-004).
    validatorUrl: null
  });
};
