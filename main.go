package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env", err)
	}

	host := os.Getenv("DB_HOST")
	Port := os.Getenv("DB_PORT")
	port, err := strconv.Atoi(Port)
	if err != nil {
		log.Fatal("Ошибка port", err)
	}
	user := os.Getenv("DB_USER")
	dbname := os.Getenv("DB_NAME")
	password := os.Getenv("DB_PASSWORD")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", // Строка подключения
		host, port, user, dbname, password)

	// Подключение к базе данных
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping() // Проверка подключения
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/users", getUser) //обьявление обработчиков
	http.HandleFunc("/users/create", CreateUser)
	http.HandleFunc("/users/delete", DeleteUser)
	log.Println("Сервер запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil) //запуск сервера

}

func getUser(w http.ResponseWriter, r *http.Request) { //обработчик GET запросов
	if r.Method != http.MethodGet {
		http.Error(w, "Ошибка метода", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.Query("SELECT id, Name, Surname FROM users")
	if err != nil {
		log.Println("Ошибка выполнения запроса:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User //срез для хранения запроса

	for rows.Next() { //обработка полей
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Surname); err != nil {
			log.Println("Ошибка сканирования строки:", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		log.Println("Ошибка после итерации:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")       //заголовок json
	if err := json.NewEncoder(w).Encode(users); err != nil { //кодирование в json+отправка
		log.Println("Ошибка кодирования JSON:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func CreateUser(w http.ResponseWriter, r *http.Request) { //обработчик CREATE
	if r.Method != http.MethodPost {
		http.Error(w, "Ошибка метода", http.StatusMethodNotAllowed)
		return
	}

	var user User //декодер запроса
	json.NewDecoder(r.Body).Decode(&user)

	if strings.TrimSpace(user.Name) == " " || strings.TrimSpace(user.Surname) == " " { //проверка NULL
		http.Error(w, "NO NULL", http.StatusBadRequest)
		return
	}

	var id int
	db.QueryRow( //выполнение запроса+возврат id
		"INSERT INTO users (Name, Surname) VALUES ($1, $2) RETURNING id",
		user.Name, user.Surname,
	).Scan(&id)
	user.ID = id

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)

}

func DeleteUser(w http.ResponseWriter, r *http.Request) { //обработчик DELETE
	if r.Method != http.MethodDelete {
		http.Error(w, "Ошибка метода", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id") //обработка запроса, определение id
	if id == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(id) //конвертация id
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM users WHERE id = $1", itemID) //выполнение запроса
	if err != nil {
		http.Error(w, "DB err", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) //успешный успех
}
