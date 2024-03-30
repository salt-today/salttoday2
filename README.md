# salttoday2

Electric Boogaloo

## Run locally

### Prerequisites

- [air](https://github.com/cosmtrek/air)
- [templ](https://github.com/a-h/templ)


### Running
- `docker-compose up mysql`
- Run the server via air - This will live reload the app on save.
  - `air`
- Run the templ generator and proxy - This will automatically generate your templ files and provide hot reloads
  - `make templ`
- Run the tailwindcss generator
  - `make tailwind`
- (Optional) Populate the database with some initial testing data via the two following options:
   - `go run localdev/db/populate.go`
   - Run the "Populate DB" run config in VS Code

