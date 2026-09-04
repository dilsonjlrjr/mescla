# Imagem do mescla-api (fasthttp) — serviço `api` do compose, contínuo (não é
# one-shot como o `builder`). Lê o banco de catálogo já seedado pelo builder
# no volume compartilhado /data; front/ chama isto pela rede (via nginx em
# produção, via proxy do Vite em dev) — wails/ nunca depende deste container,
# continua offline com bind direto ao mesmo api/service dentro do processo desktop.
FROM golang:1.25-bookworm AS build
WORKDIR /app
COPY . .
RUN go build -o /usr/local/bin/mescla-api ./api/cmd/apiserver

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /usr/local/bin/mescla-api /usr/local/bin/mescla-api

ENV MESCLA_DB_PATH=/data/paint_knowledge.db
ENV MESCLA_LISTEN_ADDR=:8080
EXPOSE 8080

ENTRYPOINT ["mescla-api"]
