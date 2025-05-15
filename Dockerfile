FROM golang:1.24.3 AS builder

ARG MIT_SERVER=${MIT_SERVER}

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 go build -o mitbot -ldflags "-X main.version=dev" ./cmd/mitbot/main.go

FROM scratch

COPY --from=builder /app/mitbot .

ENTRYPOINT ["/mitbot"]
