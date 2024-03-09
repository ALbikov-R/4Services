package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	pb "github.com/ALbikov-R/4ServicesGRPC/gen"
	"github.com/IBM/sarama"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/mgo.v2/bson"
)

type Order struct {
	ID      string     `json:"id" bson:"_id"` //Код заказа
	Data    string     `json:"data"`          //Дата Заказа
	Product []Products `json:"product"`       //Продукты
}
type Products struct {
	ItemID   string `json:"item_id"`  //Код продукта
	Name     string `json:"name"`     //Наименование
	Quantity int    `json:"quantity"` //Количество
	Price    int    `json:"price"`    //Стоимость
}
type InvProd struct {
	Quantity int    `json:"quantity"` //Количество
	Price    string `json:"price"`    //Стоимость
}
type Producer struct {
	prod    sarama.AsyncProducer
	signals chan os.Signal
}
type Cart struct {
	Products []Products `json:"product"`
}
type Message struct {
	Typemes     string `json:"typemes"`
	Description string `json:"description"`
	Date        string `json:"data"`
}

type Congrpc struct {
	client pb.InvOrdClient
	con    *grpc.ClientConn
	cancel context.CancelFunc
	ctx    context.Context
}

var (
	client         *mongo.Client
	MongoURI       = "mongodb://localhost:27017"
	PortAddr       = ":8081"
	DataBaseName   = "OrderService"
	CollectionName = "Order"
	producer       Producer
	prodAddr       = []string{"localhost:9092"}
	topicName      = "Order_test"
	connect        Congrpc
)

func main() {
	err := ConnectMongoDB()
	if err != nil {
		log.Fatal(err)
		return
	} else {
		fmt.Println("Подключение к MongoDB успешно!")
		defer client.Disconnect(context.Background())
	}
	/*
		if err = CreateProducer(); err != nil {
			log.Fatal(err)
		}*/
	ConnectGrpc()
	defer CloseProducer()
	router := mux.NewRouter()
	router.HandleFunc("/orders", GetOrders).Methods("GET")           //Получить информацию о всех заказах
	router.HandleFunc("/orders/{id}", GetOrder).Methods("GET")       //Получить информацию об заказе с номером ID
	router.HandleFunc("/orders", CreateOrder).Methods("POST")        //Создать заказ
	router.HandleFunc("/orders/{id}", KafkaMethod).Methods("POST")   //Создать заказ
	router.HandleFunc("/orders/{id}", UpdateOrder).Methods("PUT")    //Изменить в заказе ID
	router.HandleFunc("/orders/{id}", DeleteOrder).Methods("DELETE") //Удалить заказ ID

	fmt.Println("Сервер слушате порт " + os.Getenv("PORT_router"))
	http.ListenAndServe(os.Getenv("PORT_router"), router)
	CloseGrpc()
}
func CloseGrpc() {
	connect.con.Close()
	connect.cancel()
}
func ConnectGrpc() {
	var err error
	connect.con, err = grpc.Dial(os.Getenv("PORT_gRPC"), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatal(err)
	}
	connect.client = pb.NewInvOrdClient(connect.con)
	connect.ctx, connect.cancel = context.WithTimeout(context.Background(), time.Second)
}
func CloseProducer() {
	if err := producer.prod.Close(); err != nil {
		log.Fatal(err)
	}
}
func CreateProducer() error {
	config := sarama.NewConfig()
	var err error
	proder, err := sarama.NewAsyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		return err
	}
	producer.prod = proder
	producer.signals = make(chan os.Signal, 1)
	signal.Notify(producer.signals, os.Interrupt)
	return nil
}

//Замена данных заказа ID
/*
func ReplaceID(id string, prod []Products) error {
	collection := client.Database(DataBaseName).Collection(CollectionName)
	filter := bson.M{"_id": id}
	_, err := collection.ReplaceOne(context.TODO(), filter, prod)
	if err != nil {
		return err
	}
	return nil
}*/
func ReplaceID(id string, prods []Products) error {
	collection := client.Database(DataBaseName).Collection(CollectionName)
	filter := bson.M{"_id": id}
	var order Order
	err := collection.FindOne(context.TODO(), filter).Decode(&order)
	if err != nil {
		return err //Нет такого элемента в БД
	}
	order.Product = append(prods)
	_, err = collection.ReplaceOne(context.TODO(), filter, order)
	if err != nil {
		return err
	}
	return nil
}

// Вставка данных в БД
func InsertData(prods []Products) (Order, error) {
	collection := client.Database(DataBaseName).Collection(CollectionName)
	Json, _ := primitive.NewObjectID().MarshalJSON()
	var data Order
	data.ID = string(Json[1 : len(Json)-1])
	data.Data = time.Now().Format("02-01-2006 15:04:05")
	for i := 0; i < len(prods); i++ {
		r, err := connect.client.GetProduct(context.Background(), &pb.IdRequest{Id: prods[i].ItemID})
		if err != nil {
			log.Fatal(err)
		}
		var bprod InvProd = InvProd{
			Quantity: int(r.GetProd().GetQuantity()),
			Price:    r.GetProd().GetPrice(),
		}

		str := strings.Replace(bprod.Price, " руб.", "", -1)
		f, _ := strconv.Atoi(str)
		prods[i].Price = f
		prods[i].Quantity = bprod.Quantity
	}
	data.Product = append(data.Product, prods...)
	_, err := collection.InsertOne(context.TODO(), data)
	if err != nil {
		return Order{}, err
	}
	return data, nil
}

// Нахождение всех элементов в БД
func FindAll() ([]Order, error) {
	collection := client.Database(DataBaseName).Collection(CollectionName)
	cur, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())
	var orders []Order
	for cur.Next(context.TODO()) {
		var order Order
		if err := cur.Decode(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// Нахождение по одному элементу
func FindId(id string) (Order, error) {
	colletion := client.Database(DataBaseName).Collection(CollectionName)
	filter := bson.M{"_id": id}
	var order Order
	err := colletion.FindOne(context.TODO(), filter).Decode(&order)
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

// Нахождение по одному элементу
func DeleteId(id string) (int64, error) {
	colletion := client.Database(DataBaseName).Collection(CollectionName)
	filter := bson.M{"_id": id}
	res, err := colletion.DeleteOne(context.TODO(), filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}
func ConnectMongoDB() error { //Соединение с MongoDB
	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

// Получить информацию о всех заказах
func GetOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := FindAll()
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// Получить информацию об заказе с номером ID
func GetOrder(w http.ResponseWriter, r *http.Request) {
	order, err := FindId(mux.Vars(r)["id"])
	if err != nil {
		if err == mongo.ErrNoDocuments {
			w.WriteHeader(http.StatusNotFound)
			errorResponse := map[string]string{
				"error":   "Resource not found",
				"message": "The resource with the specified ID does not exist.",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errorResponse)
			return
		} else {
			log.Fatal(err)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// Создать заказ
func CreateOrder(w http.ResponseWriter, r *http.Request) {
	var cart Cart
	_ = json.NewDecoder(r.Body).Decode(&cart)
	order, err := InsertData(cart.Products)
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
func KafkaMethod(w http.ResponseWriter, r *http.Request) {
	order, err := FindId(mux.Vars(r)["id"])
	time := time.Now().Format("02-01-2006 15:04:05")
	var message *Message = &Message{Date: time}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			w.WriteHeader(http.StatusNotFound)
			errorResponse := map[string]string{
				"error":   "Resource not found",
				"message": "The resource with the specified ID does not exist.",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errorResponse)
			message.Description = "Order " + string(mux.Vars(r)["id"]) + " does not exist"
			message.Typemes = "Order not found"
			SendMessage(message)
			return
		} else {
			log.Fatal(err)
		}
	}
	message.Description = "Order " + string(mux.Vars(r)["id"]) + " exist"
	message.Typemes = "Order found"
	SendMessage(message)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
func SendMessage(message *Message) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}
	prodmessage := &sarama.ProducerMessage{
		Topic: topicName,
		Value: sarama.ByteEncoder(jsonMessage),
	}
	producer.prod.Input() <- prodmessage
	return nil
}

// Изменить в заказе ID
func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	var prods []Products
	_ = json.NewDecoder(r.Body).Decode(&prods)
	err := ReplaceID(mux.Vars(r)["id"], prods)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			w.WriteHeader(http.StatusNotFound)
			errorResponse := map[string]string{
				"error":   "Resource not found",
				"message": "The resource with the specified ID does not exist.",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errorResponse)
			return
		} else {
			log.Fatal(err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// Удалить заказ ID
func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	count, err := DeleteId(mux.Vars(r)["id"])
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		w.WriteHeader(http.StatusNotFound)
		errorResponse := map[string]string{
			"error":   "Resource not found",
			"message": "The resource with the specified ID does not exist.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errorResponse)
		//Добавить return???
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
