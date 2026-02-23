# Build Stage
FROM golang:1.26.0-alpine3.23 AS builder
WORKDIR /go-bank
COPY . .
RUN go build -o main main.go


# Run Stage
FROM alpine:3.23
WORKDIR /go-bank
COPY --from=builder /go-bank/main .
COPY app.env .

EXPOSE 8080
CMD [ "/go-bank/main" ]