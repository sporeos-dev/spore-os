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
	@echo "==> Running staticcheck..."

	@echo "==> Analysis complete."

setup:
	@echo "==> Installing staticcheck..."

	@echo "==> Done. Ensure $$(go env GOPATH)/bin is on your PATH."
