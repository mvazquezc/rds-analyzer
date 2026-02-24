# Build stage
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.23 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w \
        -X github.com/openshift-kni/rds-analyzer/internal/cli.Version=${VERSION} \
        -X github.com/openshift-kni/rds-analyzer/internal/cli.Commit=${COMMIT} \
        -X github.com/openshift-kni/rds-analyzer/internal/cli.BuildDate=${BUILD_DATE}" \
    -o /rds-analyzer \
    ./cmd/rds-analyzer

FROM scratch

COPY --from=builder /rds-analyzer /rds-analyzer

ENTRYPOINT ["/rds-analyzer"]
