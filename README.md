# GoGin-API-REST-Template
Go (Golang) API REST Template/Boilerplate with Gin Framework

[![Open Source Love](https://badges.frapsoft.com/os/mit/mit.svg?v=102)](https://github.com/ellerbrock/open-source-badge/)
![Build Status](https://travis-ci.com/antonioalfa22/GoGin-API-REST-Template.svg?branch=master)


## 1. Run with Docker

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

## 2. Generate Docs

```shell script
# Get swagger
go get -u github.com/go-swagger/go-swagger/cmd/swagger

# Generate swagger.yml
swagger generate spec -o ./api/swagger.yaml --scan-models

# Serve docs
swagger serve -F=swagger ./api/swagger.yaml
```
