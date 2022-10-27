.PHONY: format
format:
	@go install golang.org/x/tools/cmd/goimports@latest
	goimports -local "github.com/Kcrong/autostudy" -w .
	gofmt -s -w .
	go mod tidy

.PHONY: test
test:
	@go install github.com/rakyll/gotest@latest
	gotest -race -cover -v ./...

.PHONY: update
update:
	@go get -u all
	go mod tidy
