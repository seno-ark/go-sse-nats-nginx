# go-api/Dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o go-api

FROM golang:1.22
WORKDIR /app
COPY --from=builder /app/go-api .
CMD ["./go-api"]
