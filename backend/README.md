# Para levantar el backend

Para correr la aplicación se debe correr el [Docker](docker-compose.yml) que levanta una base de datos Postgres y ejecuta [el script de SQL](db/init.sql), en el mismo se van a definir las tablas que vamos a usar la aplicación.

Luego se tiene que setear las variables de entorno `NSPPSQLDS` que establece el datasource de la base de datos que va a usar la aplicación, con el formato: "user=[USUARIO] password=[CONTRASEÑA] host=[HOST] port=[PORT] dbname=[NOMBRE DE DB] sslmode=disable" Las configuraciones de las mismas se encuentran en el archivo de docker.

Por defecto se encuentra configurado el puerto del servidor HTTP a `127.0.0.1` y el puerto a `8080`, si se desea cambiar se pueden configurar las variables de entorno `NSPHOST` y `NSPPORT` respectivamente
