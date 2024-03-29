# salttoday2

Electric Boogaloo

[![UI CI](https://github.com/salt-today/salttoday2/actions/workflows/js_ci.yaml/badge.svg)](https://github.com/salt-today/salttoday2/actions/workflows/js_ci.yaml) [![Go CI Build](https://github.com/salt-today/salttoday2/actions/workflows/go.yml/badge.svg)](https://github.com/salt-today/salttoday2/actions/workflows/go.yml)

## Run locally

### Prerequisites

1. [air](https://github.com/cosmtrek/air)
2. [templ](https://github.com/a-h/templ)

1. `docker-compose up mysql`
2. Run the server via air - This will live reload the app on save.
  1. `air`
3. Run the templ generator and proxy - This will automatically generate your templ files and provide hot reloads
  1. `make templ`
3. Run the tailwindcss generator
  1. `make tailwind`
4. (Optional) Populate the database with some initial testing data via the two following options:
   1. `go run localdev/db/populate.go`
   2. Run the "Populate DB" run config in VS Code

