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

# Копируем исходники ТОЛЬКО inventory-service (и других, если нужны для сборки inventory)
# Если inventory не зависит от кода api-gateway или order-service напрямую, их можно не копировать
COPY inventory-service/ ./inventory-service/
# COPY api-gateway/ ./api-gateway/ # Раскомментируй, если нужно
# COPY order-service/ ./order-service/ # Раскомментируй, если нужно

# Устанавливаем рабочую директорию на собираемый сервис
WORKDIR /app/inventory-service

# Собираем приложение
# Вместо go build . используем go build ./cmd/main.go, если твой main файл там
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /inventory-app . # <-- ПРАВИЛЬНЫЙ ПУТЬ (собираем текущую директорию)

# --- Final Stage ---
FROM alpine:latest

WORKDIR /root/

# Копируем только скомпилированный бинарник
COPY --from=builder /inventory-app .

# Открываем порт, который слушает gRPC сервер (50051 по умолчанию)
EXPOSE 50051

# Команда для запуска приложения
CMD ["./inventory-app"]