FROM golang:1.25-alpine AS builder

WORKDIR /app

ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_DATE=unknown

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build \
    -ldflags "-X blackgrid/internal/version.Version=${VERSION} \
              -X blackgrid/internal/version.Commit=${COMMIT_SHA} \
              -X blackgrid/internal/version.BuildDate=${BUILD_DATE}" \
    -o /server cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /server .

EXPOSE 8080
CMD ["/app/server"]
