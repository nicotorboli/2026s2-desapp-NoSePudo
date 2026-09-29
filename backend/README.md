# Iniciar el servidor

 ### 1. Iniciar el Docker con la base de datos 
 Se debe correr el [Docker](docker-compose.yml) que levanta una base de datos Postgres y ejecuta [el script de SQL](db/init.sql) que define las tablas a utilizar. Los valores configurados de la db son:
 
  | Variable | Valor |
  | :-----: | :-----: |
  | Usuario | devuser | 
  | Contraseña | devpassword| 
  | Database | nsp_db | 
  | Puerto | 5433 |
 
 ### 2. Setear las variables de entorno
 Se debe establecer obligatoriamente la variable de entorno que especifica el datasource que se va a utilizar en la aplicación.
 ```env
 NSPPSQLDS="user=[USUARIO] password=[CONTRASEÑA] host=[HOST] port=[PORT] dbname=[NOMBRE DE DB] sslmode=disable"
 ``` 
 ## 3. Descargar las dependencias
 Descargar las dependencias del proyecto por medio del comando
 ```sh
 go mod tidy 
```
### 4. Levantar el servidor
Por defecto el servidor se levanta en el host 127.0.0.1 (localhost) y el puerto 8080. Se puede configurar estableciendo las variables de entorno:
 ```env
 NSPHOST=[HOST DEL SERVIDOR]
 NSPPORT=[PUERTO DEL SERVIDOR]
 ``` 
Para ejecutar el servidor se utiliza el comando
```sh
go run ./cmd/server/main.go
```

# Ejecutar los tests
## TODO
