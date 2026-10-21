# Terraform Explorer

Terraform Explorer is a local, read-only application for understanding
Terraform configuration. It parses native HCL, maps static value flow, and
connects an interactive graph to the original source code.

The application never runs Terraform, downloads modules, evaluates functions,
or sends source code to an external service.

## Run the application

Go 1.25 or newer is required. The compiled React interface is embedded in the
Go binary, so Node.js is not required for a normal run.

```sh
go run -buildvcs=false ./cmd/terraform-explorer serve
```

Open <http://127.0.0.1:8787> and enter a local directory containing `.tf`
files. To try the bundled fixture, use:

```text
./testdata/simple-module
```

The same command is available through Make:

```sh
make run
```

## Current application features

- Local path selection and rescanning.
- Native HCL parsing with partial results for malformed files.
- Resources, data sources, module calls, variables, locals, and outputs.
- Static value-flow graph with zoom, pan, minimap, ELK layout, search, and
  object-type filters.
- Incoming and outgoing dependency inspection.
- Unresolved-reference reporting.
- File navigation and a read-only, locally bundled Monaco HCL editor.
- Exact file, line, column, and byte ranges.
- Parser diagnostics linked to source.
- Same-origin API protection, source path validation, file-size limits, and
  symbolic-link rejection.

The graph is a static configuration graph, not Terraform's runtime execution
graph and not a cloud architecture topology. This version analyzes one module
directory at a time; recursive local-module discovery is the next backend
step.

## CLI inventory

The JSON scanner remains available independently of the UI:

```sh
go run -buildvcs=false ./cmd/terraform-explorer scan ./testdata/simple-module
```

Use `--pretty=false` for compact JSON.

## Build a binary

The checked-in frontend build is embedded automatically:

```sh
go build -buildvcs=false -trimpath -o terraform-explorer ./cmd/terraform-explorer
./terraform-explorer serve
```

Or rebuild both the UI and binary:

```sh
make build
```

Node.js is needed only when rebuilding the frontend:

```sh
cd web
npm ci
npm run build
```

## Development

Run the Go backend:

```sh
go run -buildvcs=false ./cmd/terraform-explorer serve
```

Then run Vite in another terminal. Requests under `/api` are proxied to the Go
process:

```sh
cd web
npm run dev
```

Open <http://127.0.0.1:5173>.

## Verification

```sh
go test ./...
go test -race ./...
go vet ./...

cd web
npm run typecheck
npm run build
npm audit --omit=dev
```
