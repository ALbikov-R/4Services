package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"

	pb "github.com/ALbikov-R/4ServicesGRPC/gen"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"google.golang.org/grpc"
)

type Product struct {
	ID       string `json:"item_id"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    string `json:"price"`
}
type Fproduct struct {
	Product
	Weight      float64
	Description string
}

var (
	db *sql.DB
)

type grpcServer struct {
	pb.UnimplementedInvOrdServer
}

func (s *grpcServer) SendProduct(ctx context.Context, in *pb.CreateRequest) (*pb.StatusReply, error) {
	var prod Product = Product{
		ID:       in.GetProd().GetId(),
		Name:     in.GetProd().GetName(),
		Quantity: int(in.GetProd().GetQuantity()),
		Price:    in.GetProd().GetPrice(),
	}
	_, err := Insert(prod)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return &pb.StatusReply{Flag: false, Message: "item already exists"}, err
		} else {
			log.Fatal(err)
		}
	}
	log.Printf("item - %s success created\n", prod.ID)

	return &pb.StatusReply{Flag: true, Message: "success created"}, nil
}
func (s *grpcServer) DelProduct(ctx context.Context, in *pb.IdRequest) (*pb.StatusReply, error) {
	count, err := DeleteID(in.GetId())
	if err != nil {
		log.Fatal(err)
	}
	if count == 0 {
		log.Printf("the resource with the specified ID %s does not exist.\n", in.GetId())
		return &pb.StatusReply{Flag: false, Message: "the resource with the specified ID does not exist."}, nil
	}
	log.Printf("item - %s success deleted\n", in.GetId())
	return &pb.StatusReply{Flag: true, Message: "success deleted"}, nil
}
func (s *grpcServer) GetProduct(ctx context.Context, in *pb.IdRequest) (*pb.GetProdReply, error) {
	prod, err := GetDataID(in.GetId())
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("item not found\n")
			return nil, fmt.Errorf("item not found")
		} else {
			log.Fatal(err)
		}
	}
	log.Printf("item - %s success sended\n", prod.ID)
	return &pb.GetProdReply{Prod: &pb.Product{
		Id:       prod.ID,
		Name:     prod.Name,
		Quantity: int32(prod.Quantity),
		Price:    prod.Price,
	}}, nil
}
func (s *grpcServer) UpdProduct(ctx context.Context, in *pb.CreateRequest) (*pb.StatusReply, error) {
	var prod Product = Product{
		ID:       in.GetProd().GetId(),
		Name:     in.GetProd().GetName(),
		Quantity: int(in.GetProd().GetQuantity()),
		Price:    in.GetProd().GetPrice(),
	}
	_, err := UpdateID(prod)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("item not found\n")
			return nil, fmt.Errorf("item not found")
		} else {
			log.Fatal(err)
		}
	}
	log.Printf("item - %s success updated\n", prod.ID)
	return &pb.StatusReply{Flag: true, Message: "success updated"}, nil
}
func main() {
	db = ConnectDd()
	defer db.Close()
	log.Println("Подключение к PostgreSQL успешно!")
	ch := make(chan error)
	go gStart(ch)
	go restStart(ch)
	for i := 0; i < 2; i++ {
		log.Println(<-ch)
	}
	fmt.Println("service is down")
}
func restStart(ch chan error) {
	router := mux.NewRouter()
	router.HandleFunc("/inventory", GETInv).Methods("GET")
	router.HandleFunc("/inventory/{id}", GETInvID).Methods("GET")
	router.HandleFunc("/inventory", CreateInv).Methods("POST")
	router.HandleFunc("/inventory/{id}", UpdInv).Methods("PUT")
	router.HandleFunc("/inventory/{id}", DelInv).Methods("DELETE")

	log.Println("Сервер слушает порт " + os.Getenv("PORT_router"))
	if err := http.ListenAndServe(os.Getenv("PORT_router"), router); err != nil {
		ch <- fmt.Errorf("failed to serve: %v", err)
	}
}
func gStart(ch chan error) {
	lis, err := net.Listen("tcp", os.Getenv("PORT_gRPC"))
	if err != nil {
		ch <- fmt.Errorf("%v", err)
		runtime.Goexit()
	}
	s := grpc.NewServer()
	pb.RegisterInvOrdServer(s, &grpcServer{})
	log.Println("server grpc is listening")
	if err := s.Serve(lis); err != nil {
		ch <- fmt.Errorf("failed to serve: %v", err)
	}
}
func GETInv(w http.ResponseWriter, r *http.Request) {
	prods := GetData()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prods)
}
func GETInvID(w http.ResponseWriter, r *http.Request) {
	prod, err := GetDataID(mux.Vars(r)["id"])
	if err != nil {
		if err != sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			log.Fatal(err)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prod)
}
func CreateInv(w http.ResponseWriter, r *http.Request) {
	var prod Product
	_ = json.NewDecoder(r.Body).Decode(&prod)
	_, err := Insert(prod)
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
	json.NewEncoder(w).Encode(prod)
}
func UpdInv(w http.ResponseWriter, r *http.Request) {
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
func DelInv(w http.ResponseWriter, r *http.Request) {
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
func ConnectDd() *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	return db
}
func Insert(item Product) (int64, error) {
	res, err := db.Exec("INSERT INTO inventory (id, naming, quantity, price) VALUES ($1,$2,$3,$4)", item.ID, item.Name, item.Quantity, item.Price)
	if err != nil {
		return -1, err
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		return rowcount, err
	}
	return rowcount, nil
}
func GetDataID(IDNAME string) (Product, error) { //Обработать ошибку после работы функции
	rows := db.QueryRow("SELECT * FROM inventory WHERE id=$1", IDNAME)
	var prod Product
	// Обработка результатов запроса
	err := rows.Scan(&prod.ID, &prod.Name, &prod.Quantity, &prod.Price)
	if err != nil {
		return Product{}, err
	}
	return prod, nil
}
func GetData() []Product {
	rows, err := db.Query("SELECT * FROM inventory")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var prod []Product
	// Обработка результатов запроса
	for rows.Next() {
		var id, name, price string
		var quantity int
		err := rows.Scan(&id, &name, &quantity, &price)
		if err != nil {
			panic(err)
		}
		prod = append(prod, Product{ID: id, Name: name, Quantity: quantity, Price: price})
	}
	return prod
}

func UpdateID(item Product) (int64, error) {
	_, err := GetDataID(item.ID)
	if err != nil {
		return -1, err
	}
	res, err := db.Exec("UPDATE inventory SET naming = $2, quantity =$3, price = $4 WHERE id=$1",
		item.ID, item.Name, item.Quantity, item.Price)
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
	res, err := db.Exec("DELETE FROM inventory WHERE id = $1", IDNAME)
	if err != nil {
		return -1, err
	}
	rowcount, err := res.RowsAffected()
	if err != nil {
		return -1, err
	}
	return rowcount, nil
}
