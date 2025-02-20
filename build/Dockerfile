FROM golang:1.22

WORKDIR /app

# Копируем только файлы модулей для лучшего кэширования
COPY go.mod go.sum ./
RUN go mod tidy


COPY . .

RUN go build -o main ./cmd/server

CMD ["/app/main"]