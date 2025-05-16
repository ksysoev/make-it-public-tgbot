FROM golang:1.24.3 AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o mitbot ./cmd/mitbot/main.go

FROM scratch

COPY --from=builder /app/mitbot .

ENTRYPOINT ["/mitbot"]
