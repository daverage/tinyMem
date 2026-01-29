FROM golang:1.25.6-bookworm AS build

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        gcc \
        g++ \
        pkg-config \
        libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -tags "fts5" -ldflags "-s -w" -o /out/tinymem ./cmd/tinymem

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/tinymem /usr/local/bin/tinymem

ENTRYPOINT ["/usr/local/bin/tinymem"]
CMD ["health"]
