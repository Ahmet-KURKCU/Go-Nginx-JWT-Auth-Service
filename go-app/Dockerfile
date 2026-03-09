# --- 1. AŞAMA: İNŞA (Builder) ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Tüm kodları kopyala
COPY . .

# go.mod dosyasını oluştur ve kütüphaneleri toparla (Eksik olan satırı ekledik!)
RUN go mod init go-app
RUN go mod tidy

# Go uygulamasını derle
RUN CGO_ENABLED=0 GOOS=linux go build -o app


# --- 2. AŞAMA: ÇALIŞTIRMA (Production) ---
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

# Derlenmiş dosyayı ve HTML şablonlarını kopyala
COPY --from=builder /app/app .
COPY --from=builder /app/templates ./templates

CMD ["./app"]