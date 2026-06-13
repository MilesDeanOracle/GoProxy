FROM node:22-bookworm-slim AS frontend-builder

WORKDIR /src/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build


FROM golang:1.25-bookworm AS backend-builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
COPY third_party/ ./third_party/
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/goproxy-webserver ./cmd/webserver/


FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        dnsutils \
        inetutils-telnet \
        iproute2 \
        iputils-ping \
        nano \
        netcat-openbsd \
        procps \
        wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=backend-builder /out/goproxy-webserver /app/goproxy-webserver
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist

RUN mkdir -p /app/configs

VOLUME ["/app/configs"]

EXPOSE 9090 1080 8080

CMD ["/app/goproxy-webserver", "-config", "/app/configs/config.yaml", "-listen", "0.0.0.0:9090", "-static", "/app/frontend/dist"]
