FROM golang:1.21-alpine AS build

COPY . /app

WORKDIR /app

RUN go install github.com/a-h/templ/cmd/templ@latest
RUN templ generate

RUN go mod download

RUN go build -o /service-bin cmd/server/main.go


# TODO figure this out.
# FROM node:20 as web
# COPY ./web /web
# WORKDIR /web
# 
# RUN npm install
# 
# RUN npx tailwindcss -c tailwind.config.js -i public/styles.css -o public/output.css


FROM alpine
EXPOSE 8080

WORKDIR /app
COPY --from=build /service-bin .
COPY --from=build /app/web web
# COPY --from=web /web ./web

CMD ["/app/service-bin"]

