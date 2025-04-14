# --- Build Stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.work .

RUN mkdir -p api-gateway inventory-service order-service

COPY api-gateway/go.mod ./api-gateway/
COPY api-gateway/go.sum ./api-gateway/

COPY inventory-service/go.mod ./inventory-service/
COPY inventory-service/go.sum ./inventory-service/

COPY order-service/go.mod ./order-service/
COPY order-service/go.sum ./order-service/

RUN go work sync

COPY inventory-service/ ./inventory-service/

WORKDIR /app/inventory-service

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /inventory-app .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /inventory-app .
EXPOSE 8081
CMD ["./inventory-app"]