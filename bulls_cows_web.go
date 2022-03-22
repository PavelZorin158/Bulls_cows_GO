package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"net/http"
)

type Users struct {
	Id    int
	Name  string
	Pas   string
	Score int
}

var users []Users
var сurName string // имя текущего пользователя, который в игре

func createTable() {
	// создание таблицы и файла БД, если их нет

	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	users_table := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "name" TEXT DEFAULT "",
        "pas" TEXT DEFAULT "",
        "score" INTEGER DEFAULT 0);`
	query, err := db.Prepare(users_table)
	if err != nil {
		log.Fatal(err)
	}
	query.Exec()
}

func index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html",
		"templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Выборка данных
	record, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	users = []Users{} //обнуляем срез
	for record.Next() {
		var user Users
		err = record.Scan(&user.Id, &user.Name, &user.Pas, &user.Score)
		if err != nil {
			panic(err)
		}
		fmt.Printf("User: %d %s %s %d\n", user.Id, user.Name, user.Pas, user.Score)
		users = append(users, user)
	}
	t.ExecuteTemplate(w, "index", users)
}

func newgame(w http.ResponseWriter, r *http.Request) {
	if сurName == "" {
		// надо залогинится
		// переходим на страницу логина
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		t.ExecuteTemplate(w, "login", nil)
	} else {
		// переходим на страницу игры
	}
}

func handleFunc() {
	http.HandleFunc("/", index)
	http.HandleFunc("/newgame", newgame)

	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static/"))))

	http.ListenAndServe(":5000", nil)
}

func main() {
	createTable()
	handleFunc()
}
