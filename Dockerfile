# Build the Go binary natively, then cross-compile for BuildKit's target platform.
FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# The official Deno binary image publishes amd64 and arm64 variants.
FROM denoland/deno:bin-2.9.8 AS deno

# Keep only the runtime dependencies and the compiled service in the final image.
FROM python:3.13.18-slim-bookworm AS runtime

RUN apt-get update \
    && apt-get install --no-install-recommends -y ffmpeg=7:5.1.9-0+deb12u1 ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt /tmp/requirements.txt
RUN python -m pip install --no-cache-dir --disable-pip-version-check --root-user-action=ignore \
    --requirement /tmp/requirements.txt \
    && rm /tmp/requirements.txt

COPY --from=build /out/server /usr/local/bin/server
COPY --from=deno /deno /usr/local/bin/deno

RUN useradd --uid 10001 --create-home app \
    && mkdir -p /data/audio \
    && chown -R app:app /data

ENV HTTP_ADDR=0.0.0.0:8080 \
    SQLITE_PATH=/data/queue.db \
    AUDIO_DIR=/data/audio

WORKDIR /app
EXPOSE 8080
USER app

ENTRYPOINT ["/usr/local/bin/server"]
