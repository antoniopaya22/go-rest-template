check-swagger:
	which swagger || (go get -u github.com/go-swagger/go-swagger/cmd/swagger)

swagger: check-swagger
	go mod vendor && swagger generate spec -o ./api/swagger.yaml --scan-models

serve-swagger: check-swagger
	swagger serve -F=swagger ./api/swagger.yaml

build:
	go build -o bin/restapi cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./test/...