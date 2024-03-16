package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/ALbikov-R/4ServicesGRPC/gen"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Product struct {
	ID          string  `json:"item_id"`
	Naming      string  `json:"name"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}
type Fproduct struct {
	Product
	Price    string `json:"price"`
	Quantity int    `json:"quantity"`
}

type Cart struct {
	Prods []Product `json:"product"`
}

var (
	cart     Cart
	PortAddr = os.Getenv("PORT_router")
	db       *sql.DB
)

var connect Congrpc

type Congrpc struct {
	client pb.InvOrdClient
	con    *grpc.ClientConn
	cancel context.CancelFunc
}

func main() {
	db = ConnectDd()
	MigrateUP()
	defer db.Close()
	fmt.Println("Подключение к PostgreSQL успешно!")

	ConnectGrpc()
	router := mux.NewRouter()
	router.HandleFunc("/products", GetProducts).Methods("GET")           //Получить информацию о всех продуктах
	router.HandleFunc("/products/{id}", GetProduct).Methods("GET")       //Получить информацию о продукте с номером ID
	router.HandleFunc("/products", CreateProduct).Methods("POST")        //Добавить продукт
	router.HandleFunc("/products/{id}", UpdateProduct).Methods("PUT")    //Изменить продукт по ID
	router.HandleFunc("/products/{id}", DeleteProduct).Methods("DELETE") //Удалить продукт ID
	router.HandleFunc("/products/{id}", AddProd).Methods("POST")
	router.HandleFunc("/cart", GETCart).Methods("GET")
	router.HandleFunc("/cart", POSTCart).Methods("POST")

	fmt.Println("Сервер слушате порт " + PortAddr)
	log.Fatal(http.ListenAndServe(PortAddr, router))
	CloseGrpc()
}
func MigrateUP() {
	m, err := migrate.New("file://migrations", DatabaseURL1()+"&x-migrations-table=inventory_schema")
	if err != nil {
		log.Fatal("Fatal to initialize migrate", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Failed to apply migrations:", err)
	}
	log.Println("Migrations applied successfully")
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
}

func Insert(item Product) (int64, error) {
	res, err := db.Exec("INSERT INTO items (id, naming, weight, description) VALUES ($1,$2,$3,$4)", item.ID, item.Naming, item.Weight, item.Description)
	if err != nil {
		return -1, err
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		return rowcount, err
	}
	return rowcount, nil
}
func UpdateID(item Product) (int64, error) {
	_, err := GetDataID(item.ID)
	if err != nil {
		return -1, err
	}
	res, err := db.Exec("UPDATE items SET naming = $2, weight =$3, description = $4 WHERE id=$1",
		item.ID, item.Naming, item.Weight, item.Description)
	if err != nil {
		return -1, err
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		return -1, err
	}
	return rowcount, nil
}
func DeleteID(IDNAME string) (int64, error) {
	res, err := db.Exec("DELETE FROM items WHERE ID = $1", IDNAME)
	if err != nil {
		return -1, err
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		return -1, err
	}
	return rowcount, nil
}
func GetDataID(IDNAME string) (Product, error) { //Обработать ошибку после работы функции
	rows := db.QueryRow("SELECT * FROM items WHERE id=$1", IDNAME)
	var prod Product
	// Обработка результатов запроса
	err := rows.Scan(&prod.ID, &prod.Naming, &prod.Weight, &prod.Description)
	if err != nil {
		return Product{}, err
	}
	return prod, nil
}
func GetData() []Product {
	rows, err := db.Query("SELECT * FROM items")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var prod []Product
	// Обработка результатов запроса
	for rows.Next() {
		var id, naming, description string
		var weight float64
		err := rows.Scan(&id, &naming, &weight, &description)
		if err != nil {
			panic(err)
		}
		prod = append(prod, Product{ID: id, Naming: naming, Weight: weight, Description: description})
	}
	return prod
}
func ConnectDd() *sql.DB {
	var err error
	for {
		db, err = sql.Open("postgres", DatabaseURL1())
		if err != nil {
			log.Println(err)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	for {
		check := db.Ping()
		if check != nil {
			log.Println(check)
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}

	return db
}
func DatabaseURL1() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}
func DatabaseURL() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
}
func GetProducts(w http.ResponseWriter, r *http.Request) {
	prods := GetData()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prods)
}
func GetProduct(w http.ResponseWriter, r *http.Request) {
	prod, err := GetDataID(mux.Vars(r)["id"])
	if err != nil {
		if err == sql.ErrNoRows {
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
	json.NewEncoder(w).Encode(prod)
}
func CreateProduct(w http.ResponseWriter, r *http.Request) {
	fmt.Println("here")
	var prod Fproduct
	_ = json.NewDecoder(r.Body).Decode(&prod)
	_, err := Insert(prod.Product)
	fmt.Println(prod)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			w.WriteHeader(http.StatusNotFound)
			errorResponse := map[string]string{
				"error":   "Product is already exist",
				"message": "The resource with the specified ID already exist.",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(errorResponse)
			return
		} else {
			log.Fatal(err)
		}
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prod.Product)
	status, err := connect.client.SendProduct(context.Background(), &pb.CreateRequest{
		Prod: &pb.Product{
			Id:       prod.ID,
			Name:     prod.Naming,
			Quantity: int32(prod.Quantity),
			Price:    prod.Price,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(status.Message)
}
func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	var prod Product
	_ = json.NewDecoder(r.Body).Decode(&prod)
	_, err := UpdateID(prod)
	if err != nil {
		if err == sql.ErrNoRows {
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
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func POSTCart(w http.ResponseWriter, r *http.Request) {
	jsonData, _ := json.Marshal(cart)
	fmt.Println(string(jsonData))
	resp, err := http.Post("http://order:8081/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	w.WriteHeader(http.StatusNoContent)
}
func GETCart(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart.Prods)
}
func AddProd(w http.ResponseWriter, r *http.Request) {
	prod, err := GetDataID(mux.Vars(r)["id"])
	if err != nil {
		if err == sql.ErrNoRows {
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
	cart.Prods = append(cart.Prods, prod)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart.Prods)
}
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	count, err := DeleteID(mux.Vars(r)["id"])
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
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
