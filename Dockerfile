FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o helm-cel ./cmd/helm-cel

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/helm-cel /usr/local/bin/

ENTRYPOINT ["helm-cel"]