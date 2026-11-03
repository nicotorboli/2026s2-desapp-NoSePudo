-include app.env
export 

.PHONY: run test

run:
	go run cmd/server/main.go


test:
	go test -v -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
