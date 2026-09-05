# syntax=docker/dockerfile:1

# --- build stage ---
# Cross-compila um binário estático. Sem CGO, o binário final não
# depende de nenhuma lib do sistema além do que o D-Bus/BlueZ já provê
# via socket (que é montado em runtime, não linkado).
FROM golang:1.26-bookworm AS builder

WORKDIR /src

# go.sum precisa existir antes do primeiro build — rode `go mod tidy`
# localmente (com internet liberada) uma vez e comite o go.sum gerado.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=arm64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# --- runtime stage ---
# distroless/static: sem shell, sem gerenciador de pacotes, só o
# binário + certificados. Isso é o que mantém a imagem pequena e a
# pegada de memória mínima no Orange Pi.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/worker /worker

ENV CONFIG_PATH=/data/config.yaml
ENV VAULT_DIR=/data/vault

ENTRYPOINT ["/worker"]
