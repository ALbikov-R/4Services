# Используем официальный образ Golang в качестве базового образа
FROM golang:1.22-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /go/src/order

COPY go.mod .
COPY go.sum .
RUN go mod download

# Копируем исходный код внутрь контейнера
COPY . .

# Компилируем Go-приложение в бинарный файл
RUN go build -o /go/bin/order .

# Создаем конечный образ
FROM alpine AS runner

# Копируем бинарный файл из предыдущего образа в текущий образ
COPY --from=builder /go/bin/order /order

# Задаем рабочую директорию
WORKDIR /
EXPOSE 8081
# Запускаем приложение
CMD ["/order"]
