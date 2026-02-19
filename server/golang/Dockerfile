# LOCAL DEVELOPMENT ONLY — uses air for hot reload.
# Do NOT use this in production or perf tests.
# To build a production binary: go build -o server ./cmd/server

FROM golang:1.25-alpine

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8080

CMD ["air"]
