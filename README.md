# salttoday2

Electric Boogaloo

[![UI CI](https://github.com/salt-today/salttoday2/actions/workflows/js_ci.yaml/badge.svg)](https://github.com/salt-today/salttoday2/actions/workflows/js_ci.yaml) [![Go CI Build](https://github.com/salt-today/salttoday2/actions/workflows/go.yml/badge.svg)](https://github.com/salt-today/salttoday2/actions/workflows/go.yml)

## Run locally under VS Code

1. `docker-compose up mysql`
2. Run the server via the two following options:
   1. `go run cmd/http/main.go`
   2. Launch the "Run Server" run config in VS Code
3. (Optional) Populate the database with some initial testing data via the two following options:
   1. `go run localdev/db/populate.go`
   2. Run the "Populate DB" run config in VS Code

## Run locally under docker compose

1. `docker compose up --build`
