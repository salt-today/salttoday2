FROM golang:1.21-alpine AS build

COPY . /app

WORKDIR /app

RUN go mod download

RUN go build -o /scraper-bin cmd/scraper/main.go

FROM alpine

WORKDIR /app
COPY --from=build /scraper-bin .

CMD ["/app/scraper-bin"]

