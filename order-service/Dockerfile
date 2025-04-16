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

# Копируем исходники order-service И inventory-service (т.к. order импортирует pb из inventory)
COPY order-service/ ./order-service/
COPY inventory-service/ ./inventory-service/
# COPY api-gateway/ ./api-gateway/ # Не нужно, если нет прямого импорта

# Устанавливаем рабочую директорию на собираемый сервис
WORKDIR /app/order-service

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /order-app .

# --- Final Stage ---
FROM alpine:latest

WORKDIR /root/

# Копируем только скомпилированный бинарник
COPY --from=builder /order-app .

# Открываем порт, который слушает gRPC сервер (50052 по умолчанию)
EXPOSE 50052

# Команда для запуска приложения
CMD ["./order-app"]