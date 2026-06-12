# Builds any service in cmd/ — select with: --build-arg SERVICE=api
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${SERVICE}

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
