# Используем официальный образ Golang в качестве базового образа
FROM golang:1.22-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /go/src/inventory

COPY go.mod .
COPY go.sum .
RUN go mod download

# Копируем исходный код внутрь контейнера
COPY . .

# Компилируем Go-приложение в бинарный файл
RUN go build -o /go/bin/inventory .

# Создаем конечный образ
FROM alpine AS runner
# Задаем рабочую директорию
WORKDIR /
# Копируем бинарный файл из предыдущего образа в текущий образ
COPY --from=builder /go/bin/inventory /inventory
# Копируем миграции БД
COPY migrations /migrations


EXPOSE 8082 1487

# Запускаем приложение
CMD ["/inventory"]
