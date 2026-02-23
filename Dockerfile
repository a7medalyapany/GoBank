# Build Stage
FROM golang:1.26.0-alpine3.23 AS builder
WORKDIR /go-bank
COPY . .
RUN go build -o main main.go
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz | tar xvz


# Run Stage
FROM alpine:3.23
WORKDIR /go-bank
COPY --from=builder /go-bank/main .
COPY --from=builder /go-bank/migrate ./migrate
COPY app.env .
COPY start.sh .
COPY db/migration ./migration

EXPOSE 8080
CMD [ "/go-bank/main" ]
ENTRYPOINT [ "/go-bank/start.sh" ]