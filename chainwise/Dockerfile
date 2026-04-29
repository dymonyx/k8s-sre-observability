# syntax=docker/dockerfile:1
FROM golang:1.26.2-alpine AS builder

ARG SERVICE=frontend
WORKDIR /src

COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/service /app/service
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/service"]
