.PHONY: lint test run
SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c

lint:
	golangci-lint run ./...

test:
	go test ./...

run:
	@ go run main.go -debug > /tmp/provider.log 2>&1 &
	@ echo "Provider is running..."

stop:
	@ pkill -f "go run main.go -debug"
	@ echo "Provider stopped."

apply:
	@value=$$(grep -m1 'TF_REATTACH_PROVIDERS=' /tmp/provider.log | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//'); \
	if [ -z "$$value" ]; then \
		echo "TF_REATTACH_PROVIDERS not found in /tmp/provider.log. Run 'make run' first."; \
		exit 1; \
	fi; \
	value="$${value#TF_REATTACH_PROVIDERS=}"; \
	value="$${value#\'}"; \
	value="$${value%\'}"; \
	export TF_REATTACH_PROVIDERS="$$value"; \
	tofu -chdir=terraform apply

destroy:
	@value=$$(grep -m1 'TF_REATTACH_PROVIDERS=' /tmp/provider.log | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//'); \
	if [ -z "$$value" ]; then \
		echo "TF_REATTACH_PROVIDERS not found in /tmp/provider.log. Run 'make run' first."; \
		exit 1; \
	fi; \
	value="$${value#TF_REATTACH_PROVIDERS=}"; \
	value="$${value#\'}"; \
	value="$${value%\'}"; \
	export TF_REATTACH_PROVIDERS="$$value"; \
	tofu -chdir=terraform destroy