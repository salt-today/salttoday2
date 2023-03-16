FROM golang:1.19-alpine AS build

COPY . /app

WORKDIR /app

RUN go mod download

RUN go build -o /service-bin cmd/http/main.go

FROM alpine

EXPOSE 3000

WORKDIR /app
COPY --from=build /service-bin .

CMD ["/app/service-bin"]
