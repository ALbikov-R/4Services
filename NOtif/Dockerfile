# Используем официальный образ Golang в качестве базового образа
FROM golang:1.22-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /go/src/notif

COPY go.mod .
COPY go.sum .
RUN go mod download

# Копируем исходный код внутрь контейнера
COPY . .

# Компилируем Go-приложение в бинарный файл
RUN go build -o /go/bin/notif .

# Создаем конечный образ
FROM alpine AS runner

# Копируем бинарный файл из предыдущего образа в текущий образ
COPY --from=builder /go/bin/notif /notif

# Задаем рабочую директорию
WORKDIR /

# Запускаем приложение
CMD ["/notif"]
