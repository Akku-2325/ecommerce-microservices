# --- Build Stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем go.work и все go.mod/go.sum для кэширования зависимостей
COPY go.work go.work.sum ./
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
COPY inventory-service/go.mod inventory-service/go.sum ./inventory-service/
COPY order-service/go.mod order-service/go.sum ./order-service/

# Скачиваем зависимости для всех модулей в рабочей области
#RUN go work download

# Копируем исходники api-gateway, inventory-service, order-service (т.к. gateway импортирует pb из обоих)
COPY api-gateway/ ./api-gateway/
COPY inventory-service/ ./inventory-service/
COPY order-service/ ./order-service/

# Устанавливаем рабочую директорию на собираемый сервис
WORKDIR /app/api-gateway

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /gateway-app .

# --- Final Stage ---
FROM alpine:latest

WORKDIR /root/

# Копируем только скомпилированный бинарник
COPY --from=builder /gateway-app .

# Открываем порт, который слушает Gin (8080 по умолчанию)
EXPOSE 8080

# Команда для запуска приложения
CMD ["./gateway-app"]