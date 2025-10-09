# syntax=docker/dockerfile:1

FROM golang:1.24 AS dependencies

COPY go.mod go.sum /modules/
WORKDIR /modules

RUN go mod download

RUN go install github.com/a-h/templ/cmd/templ@latest

# Download Tailwind CSS v3.4.3 standalone binary
# IMPORTANT: Must match version in package.json
ARG TAILWIND_VERSION=v3.4.3
RUN apt-get update && apt-get install -y --no-install-recommends curl && \
    echo "Downloading Tailwind CSS ${TAILWIND_VERSION}" && \
    curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-x64 && \
    chmod +x tailwindcss-linux-x64 && \
    mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss && \
    apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

FROM golang:1.24 AS builder

COPY --from=dependencies /go/bin/templ /go/bin/templ
COPY --from=dependencies /usr/local/bin/tailwindcss /usr/local/bin/tailwindcss

WORKDIR /workdir

COPY . /workdir

RUN /go/bin/templ generate

# Verify Tailwind version and generate CSS
RUN echo "=== Tailwind CSS Version ===" && \
    tailwindcss --help | head -2 && \
    echo "=== Generating CSS ===" && \
    tailwindcss -c ./tailwind.config.js -i ./public/styles.css -o ./public/output.css --minify && \
    echo "=== CSS Generated ===" && \
    ls -lh ./public/output.css

RUN go mod download && \
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /bin/service-bin cmd/server/main.go

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S -D -G nonroot -H -h /app nonroot && \
    mkdir -p /app && \
    chown -R nonroot:nonroot /app

COPY --from=builder /bin/service-bin /app/service-bin
COPY --from=builder /workdir/public /app/public

USER nonroot:nonroot
WORKDIR /app
ENV HOME=/tmp

EXPOSE 8080

CMD ["/app/service-bin"]
