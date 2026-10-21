.PHONY: run build ui test

run:
	go run -buildvcs=false ./cmd/terraform-explorer serve

build: ui
	go build -buildvcs=false -trimpath -o terraform-explorer ./cmd/terraform-explorer

ui:
	cd web && npm ci && npm run build

test:
	go test ./...
	cd web && npm run typecheck

