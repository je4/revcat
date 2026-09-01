FROM golang:1.27 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/revcat ./cmd/revcat

FROM alpine AS runner

COPY --from=builder /out/revcat /app/revcat

RUN mkdir ./cache
COPY data/ ./data/
COPY tools/ ./tools/

WORKDIR /app
EXPOSE 8441
ENTRYPOINT ["./revcat", "-config", "/config/revcat.toml"]
