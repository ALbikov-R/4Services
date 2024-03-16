# Интернет магазин
Проект, написанный на golang, который реализизован с помощью микросервисовной архитектуры. Имитирующая работу интернет-магазина. 
## Стек-технологий
Стек сервиса состоит:
+ Golang
+ PostgreSQL
+ MongoDB
+ Kafka
+ gRPC
+ Docker
+ REST
## Установка и запуск
Для установки сервиса:
```text
git clone https://github.com/ALbikov-R/4Services .
```
Запуск сервиса:
```text
docker-compose up --build
```
В результате чего будет запущен в Docker'e образы реализованных сервисов.
## gRPC
Proto файл расположен по адресу https://github.com/ALbikov-R/4ServicesGRPC
```text
syntax = "proto3";

option go_package = "github.com/ALbikov-R/4ServicesGRPC/gen";

package InvOrd;

message Product {
    string id = 1;
    string name = 2;
    int32 quantity = 3;
    string price = 4; 
}

message CreateRequest{
    Product prod = 1;
}
message StatusReply {
    bool flag =1;
    string message =2;
}
message IdRequest {
    string id=1;
}
message GetProdReply {
    Product prod =1;
}
service InvOrd {
    rpc SendProduct (CreateRequest) returns (StatusReply){}
    rpc DelProduct (IdRequest) returns (StatusReply) {}
    rpc GetProduct (IdRequest) returns (GetProdReply){}
    rpc UpdProduct (CreateRequest) returns (StatusReply){}
}
```
## Сервисы
Созданный список сервисов:
+ Inventory
+ Order
+ Notification
+ Product
## Prodcut service
В данном сервисе используется PostgreSQL, REST, gRPC, migrations.

### База данных PostgreSQL:
```text
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    naming varchar(255) NOT NULL,
    weight FLOAT NOT NULL,
    description varchar(255) NOT NULL
);
```
### End points
```text
	localhost:8080/product      -   GET Получить информацию о всех продуктах
    localhost:8080/product/{id} -   GET Получить информацию о продукте с номером ID
	localhost:8080/product      -   POST Добавить продукт в БД и по gRPC в Order 
	localhost:8080/product/{id} -   PUT Изменить продукт по ID
    localhost:8080/product/{id} -   DELETE Удалить продукт ID
    localhost:8080/product/{id} -   POST Добавить предмет в корзину
    localhost:8080/cart         -   GET Получить список корзины 
    localhost:8080/cart  -   POST Передать корзину в Order (orders:8081/orders POST)
```
## Order service
В данном сервисе используется REST, gRPC, migrations, Kafka, MongoDB
### End points
```text
	localhost:8081/orders      -   GET Получить информацию о всех заказах
    localhost:8081/orders/{id} -   GET Получить информацию об заказе с номером ID
	localhost:8081/orders      -   POST Создать заказ
	localhost:8081/orders/{id} -   PUT Изменить в заказе ID
    localhost:8081/orders/{id} -   DELETE Удалить заказ ID
    localhost:8081/orders/{id} -   POST Отправить уведомление в сервис Notification
```
## Notification service
Сервис, который получает уведомление о созданном заказе, используя брокер сообщения Kafka в связке с MongoDB.
### URI Kafka ссылка
```text
	kafka:9092
```
### Содержание сообщения
```text
type Message struct {
	Typemes     string `json:"typemes"`
	Description string `json:"description"`
	Date        string `json:"data"`
}
```
Typemes     - статус уведомления.
Descroption - описание уведомления.
Date        - дата уведомления
## Inventory service
В данном сервисе используется PostgreSQL, REST, gRPC, migrations. Является gRPC - сервером для сервисов Order и Product.
### База данных PostgreSQL:
```text
CREATE TABLE IF NOT EXISTS inventory (
    id SERIAL PRIMARY KEY,
    naming VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    price VARCHAR(255) NOT NULL
);
```
### End points
```text
	router.HandleFunc("/inventory", GETInv).Methods("GET")
	router.HandleFunc("/inventory/{id}", GETInvID).Methods("GET")
	router.HandleFunc("/inventory", CreateInv).Methods("POST")
	router.HandleFunc("/inventory/{id}", UpdInv).Methods("PUT")
	router.HandleFunc("/inventory/{id}", DelInv).Methods("DELETE")
	localhost:8082/inventory      -   GET Получить информацию о предметах
    localhost:8082/inventory/{id} -   GET Получить информацию о предмете с номером ID
	localhost:8082/inventory      -   POST Добавить предмет в БД 
	localhost:8082/inventory/{id} -   PUT Изменить предмет по ID
    localhost:8082/inventory/{id} -   DELETE Удалить предмет ID
```
## Example of usage 1
### Product
Используя Postman добавим предмет по ссылке localhost:8080/products по методу POST, слеудющий продукт с помощью JSON файла:
```text
{
    "item_id":"1",
    "name":"gphone",
    "weight":0.52,
    "description":"Phone",
    "price":"12500 руб.",
    "quantity":5
}
```
Используя GET метод по этой ссылке получим:
```text
[{"item_id":"1","name":"gphone","weight":0.52,"description":"Phone"}]
```
Добавим в корзину предмет по ссылке localhost:8080/products/1 (POST) и проверим корзину по ссылке localhost:8080/cart (GET), получим следующее:
```text
[{"item_id":"1","name":"gphone","weight":0.52,"description":"Phone"}]
```
Опубликуем корзину по ссылке localhost:8080/cart (POST), в результате чего данные о заказе отправятся в сервис Order и получим статус 204.
### Order
Посмотреть все заказы по ссылке localhost:8081/orders (GET)
```text
[{"id":"65f6142530646341eeaa9481","data":"16-03-2024 21:50:29","product":[{"item_id":"1","name":"gphone","quantity":5,"price":12500}]}]
```
Просмотр заказа по id localhost:8081/orders/65f6142530646341eeaa9481
```text
{"id":"65f6142530646341eeaa9481","data":"16-03-2024 21:50:29","product":[{"item_id":"1","name":"gphone","quantity":5,"price":12500}]}
```
Отправить уведомление в сервис Notification по ссылке localhost:8081/orders/65f6142530646341eeaa9481 (POST) и получим статус 200.
### Notification
В результате прошлого запроса получим в stdout 
```text
2024/03/16 21:54:54 Received message: {Order found Order 65f6142530646341eeaa9481 exist 16-03-2024 21:54:54}
```
### Inventory
Используя GET метод по ссылке localhost:8082/inventory получим:
```text
[{"item_id":"1","name":"gphone","quantity":5,"price":"12500 руб."}]
```
