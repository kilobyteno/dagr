# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# TARGETOS/TARGETARCH are set by BuildKit to match the image platform
# (e.g. linux/arm64 on Apple Silicon). Do not hardcode amd64.
ARG TARGETOS=linux
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/dagr ./cmd/server && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/dagr-worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/dagr-migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /out/dagr /dagr
COPY --from=builder /out/dagr-worker /dagr-worker
COPY --from=builder /out/dagr-migrate /dagr-migrate

USER nonroot:nonroot

EXPOSE 8383

ENTRYPOINT ["/dagr"]
