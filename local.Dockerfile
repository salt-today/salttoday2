FROM golang:1.21-alpine AS build

COPY . /app

WORKDIR /app

RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate

RUN go mod download

RUN go build -o /service-bin cmd/http/main.go
RUN go build -o /scraper-bin cmd/scraper/main.go

FROM alpine
EXPOSE $PORT

WORKDIR /app
COPY --from=build /service-bin .
COPY --from=build /scraper-bin .

CMD ["/app/service-bin"]

