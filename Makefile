SPORED_DIR := spored

.DEFAULT_GOAL := check

.PHONY: check analyze setup

check:
	@echo "==> Cleaning build and test cache..."
	@cd $(SPORED_DIR) && go clean -cache -testcache
	
	@echo "==> Building..."
	@cd $(SPORED_DIR) && go build ./...
	
	@echo "==> Vetting..."
	@cd $(SPORED_DIR) && go vet ./...
	
	@echo "==> Testing..."
	@cd $(SPORED_DIR) && go test ./...
	
	@echo "==> All checks passed."

analyze:
	@echo "==> Running go vet..."
	@cd $(SPORED_DIR) && go vet ./...

	@echo "==> Running tests with race detector..."
	@cd $(SPORED_DIR) && go test -race ./...

	@echo "==> Generating coverage report..."
	@cd $(SPORED_DIR) && go test -coverprofile=coverage.out ./...
	@cd $(SPORED_DIR) && go tool cover -func=coverage.out

	@echo "==> Running staticcheck..."
	@cd $(SPORED_DIR) && staticcheck ./...

	@echo "==> Running golangci-lint..."
	@cd $(SPORED_DIR) && golangci-lint run ./...

	@echo "==> Running govulncheck..."
	@cd $(SPORED_DIR) && govulncheck ./...

	@echo "==> Analysis complete."

setup:
	@echo "==> Installing staticcheck..."
	@go install honnef.co/go/tools/cmd/staticcheck@latest

	@echo "==> Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

	@echo "==> Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest

	@echo "==> Done. Ensure $$(go env GOPATH)/bin is on your PATH."
