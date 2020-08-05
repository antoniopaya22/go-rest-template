# GoGin-API-REST-Template
Go (Golang) API REST Template/Boilerplate with Gin Framework

[![Open Source Love](https://badges.frapsoft.com/os/mit/mit.svg?v=102)](https://github.com/ellerbrock/open-source-badge/)
[!Travis](https://travis-ci.com/antonioalfa22/GoGin-API-REST-Template.svg?branch=master)

## 1. Estructura y Flujo

```bash
├───controllers
├───models
├───middlewares
├───repository
├───routes
├───services
```

### 1.1. Models

Representa el modelo de datos, (por ejemplo un usuario).

### 1.2. Repository

Se encargan de proporcionar los métodos de acceso a base de datos para trabajar con los modelos (Entidades).

### 1.3. Middlewares

Son los componentes encargados de comprobar si se debe o no seguir con la petición. Por ejemplo autorización o roles.

### 1.4. Controllers

Los controladores son los encargados de realizar las operaciones requeridas por la petición definida en la ruta.

### 1.5. Services

Los servicios proporcionan métodos útiles compartidos por el resto de la aplicación.

_______

## 2. Ejecutar

```shell script
go build main.go
./main.exe
```

## 3. Ejecutar con Docker

1. **Build**

```shell script
make build
docker build . -t api-rest
```

2. **Run**

```shell script
docker run -p 3000:3000 api-rest
```

_______

## 4. Generate Docs

```shell script
# Get swagger
go get -u github.com/go-swagger/go-swagger/cmd/swagger

# Generate swagger.yml
swagger generate spec -o ./swagger.yaml --scan-models

# Serve docs
swagger serve -F=swagger swagger.yaml
```
