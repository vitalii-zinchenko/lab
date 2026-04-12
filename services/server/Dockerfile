# LOCAL DEVELOPMENT ONLY
# To build a production binary: go build -o server ./cmd/server

FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o ./server ./cmd/server

EXPOSE 8080

CMD ["./server"]
