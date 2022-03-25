package main

import (
	crypto "crypto/rand"
	"database/sql"
	"fmt"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
)

const maxN = 10 // максимальное количество знаков

type Users struct {
	Id    int
	Name  string
	Pas   string
	Score int
}

type UserDate struct {
	// все, что хранится в сессии
	CurName string // имя текущего пользователя, который в игре
	N       int    // кол-во цивр в звгвдвнном числе
	ZZ      string // загаданное число
	//Zag     []string // срез для загаданного числа
	Popytka int
}

type GameType struct {
	// для передачи в форму game
	Name    string
	Score   int
	Popytka int
	Sl      []string // срез для вывода резудьтатов
	Com     string   // для вывода коментария в шаблон
}

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
var users []Users
var Player Users
var CurName string // имя текущего пользователя, который в игре
var N int          // кол-во цивр в звгвдвнном числе
var ZZ string      // загаданное число
var A []string     // срез для ввода попыток
var Zag []string   // срез для загаданного числа
var M []string     // временный срез
var Popytka int
var Sl []string // история вводов
var Game GameType

func NewCryptoRand() int64 {
	// генерирует случайное число
	safeNum, err := crypto.Int(crypto.Reader, big.NewInt(100234))
	if err != nil {
		panic(err)
	}
	return safeNum.Int64()
}

func RandN(n int) string {
	// генерирует случайное число n знаков в текстовом виде
	var num string
	for i := 0; i < n; i++ {
		s := NewCryptoRand() % 10
		num += fmt.Sprint(s)
	}
	return num
}

func countCow(sl []string, s string) int {
	// находит количество s символов в срезе
	// принимает срез и символ
	// выдает количество
	res := 0
	for i := 0; i < len(sl); i++ {
		if sl[i] == s {
			res++
		}
	}
	return res
}

func indexCow(sl []string, s string) int {
	// находит первый индекс в срезе с коровой
	// принимает срез и символ
	// выдает индекс первого s символа в срезе, если нет, то -1
	for i := 0; i < len(sl); i++ {
		if sl[i] == s {
			return i
		}
	}
	return -1
}

func createTableDB() {
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

func addUserDB(name string, pas string) {
	// добавляем нового пользователя в DB
	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	records := `INSERT INTO users(name, pas) VALUES (?, ?)`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(name, pas)
	if err != nil {
		log.Fatal(err)
	}
}

func verifUserDB(name string) string {
	// возвращает пароль пользователя name
	var pas string

	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT pas FROM users WHERE name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	for record.Next() {
		record.Scan(&pas)
	}
	return pas
}

func scoreUserDB(name string) int {
	// возвращает score для пользователя name
	var scor int

	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT score FROM users WHERE name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	for record.Next() {
		record.Scan(&scor)
	}
	return scor
}

func addScoreDB(name string, scor int) int {
	// добавляет у пользователя name  в БД к score, scor и перезаписывает
	// возвращает итоговый score
	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	score := scoreUserDB(name) + scor
	records := `UPDATE users SET score = ? WHERE name LIKE ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(score, name)
	if err != nil {
		log.Fatal(err)
	}
	return score
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
	users := []Users{} //обнуляем срез
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
	session, _ := store.Get(r, "session-name")
	curName, ok := session.Values["CurName"]
	if !ok {
		// надо залогинится
		// переходим на страницу логина
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Println("ошибка создания сборки темплейта")
			fmt.Fprintf(w, err.Error())
			panic(err)
		}
		err = t.ExecuteTemplate(w, "login", "0")
		if err != nil {
			fmt.Println("ошибка запуска темплейта")
			fmt.Fprintf(w, err.Error())
			panic(err)
		}
	} else {
		// переходим на страницу игры
		t, err := template.ParseFiles("templates/kolcif.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Player = Users{}
		Player.Name = fmt.Sprint(curName)
		Player.Score = scoreUserDB(Player.Name)
		Player.Id = 0 // чтоб не выводить в шаблон коментарий об ошибке
		t.ExecuteTemplate(w, "kolcif", Player)
	}
} //sesOK

func verif_user(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	inPas := r.FormValue("password")
	fmt.Println("login.html введено имя:", name, ", пароль: ", inPas)

	pas := verifUserDB(name)
	if inPas != pas || name == "" || inPas == "" {
		fmt.Println("не совпал пароль: ", inPas, " и ", pas)
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		t.ExecuteTemplate(w, "login", "1")
	} else {
		// АВТОРИЗУЕМ ПОЛЬЗОВАТЕЛЯ
		session, _ := store.Get(r, "session-name")
		session.Values["CurName"] = name
		err := session.Save(r, w)
		if err != nil {
			fmt.Fprintln(w, "ошибка записи в сессию в func verif_user", err.Error())
			return
		}
		fmt.Println(CurName, "успешно залогинен")
		t, err := template.ParseFiles("templates/kolcif.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Player = Users{}
		Player.Name = name
		Player.Score = scoreUserDB(Player.Name)
		Player.Id = 0
		t.ExecuteTemplate(w, "kolcif", Player)
	}
} //sesOK

func addNewUser(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	inPas := r.FormValue("password")
	fmt.Println("Создание нового пользователя!")
	fmt.Println("login.html введено имя:", name, ", пароль: ", inPas)
	pas := verifUserDB(name)
	if pas != "" {
		fmt.Println("пользователь: ", name, " уже существует ")
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		err = t.ExecuteTemplate(w, "login", "2")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
	} else if name == "" || inPas == "" {
		fmt.Println("пустое имя или пароль")
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		t.ExecuteTemplate(w, "login", "3")

	} else {
		fmt.Println("создаем нового пользователя в БД")
		addUserDB(name, inPas)
		CurName = name
		t, err := template.ParseFiles("templates/kolcif.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Player = Users{}
		Player.Name = CurName
		Player.Score = scoreUserDB(CurName)
		Player.Id = 0
		t.ExecuteTemplate(w, "kolcif", Player)
	}
}

func new_user(w http.ResponseWriter, r *http.Request) {
	// УДАЛЯЕМ АВТОРИЗАЦИЮ
	session, _ := store.Get(r, "session-name")
	delete(session.Values, "CurName")
	err := session.Save(r, w)
	if err != nil {
		fmt.Fprintln(w, "ошибка обновления сессии в func new_user", err.Error())
		return
	}
	t, err := template.ParseFiles("templates/login.html",
		"templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	t.ExecuteTemplate(w, "login", "0")
} //sesOK

func new_game(w http.ResponseWriter, r *http.Request) {
	kol := r.FormValue("kol")
	session, _ := store.Get(r, "session-name")
	fmt.Println("введено цифр для начала игры: ", kol)
	n, err := strconv.Atoi(kol)
	if err != nil || n <= 0 || n > maxN {
		// не правильно введено кол-во цифр для ночала игры
		fmt.Println("Не правильное кол-во цифр!")
		t, err := template.ParseFiles("templates/kolcif.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Player = Users{}
		curName, _ := session.Values["CurName"]
		Player.Name = fmt.Sprint(curName)
		Player.Score = scoreUserDB(Player.Name)
		Player.Id = -1 // для вывода сообщения об ошибке в шаблоне
		t.ExecuteTemplate(w, "kolcif", Player)
	} else {
		// НАЧИНАЕМ ИГРУ
		zz := RandN(n)
		session.Values["ZZ"] = zz         // загаданное число в string
		session.Values["N"] = n           // кол-во цифр в загаданном числе
		session.Values["Sl"] = []string{} // обнуляем срез попыток
		session.Values["Popytka"] = 1
		err := session.Save(r, w)
		if err != nil {
			fmt.Fprintln(w, "ошибка обновления сессии в func new_game", err.Error())
			return
		}
		//A = make([]string, n)   // срез для ввода попыток
		//Zag = make([]string, n) // срез для загаданного числа
		//M = make([]string, n)   // временный срез
		//for i, s := range ZZ {
		//	Zag[i] = string(s)
		//	M[i] = string(s)
		//}
		//Sl = []string{} // обнуляем срез попыток
		fmt.Println("загадано число : ", zz)
		t, err := template.ParseFiles("templates/game.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Game.Name = fmt.Sprint(session.Values["CurName"])
		Game.Score = scoreUserDB(Game.Name)
		Game.Popytka = 1
		Game.Com = "0"
		Game.Sl = []string{}

		t.ExecuteTemplate(w, "game", Game)
	}

} //sesOK

func app(w http.ResponseWriter, r *http.Request) {
	var temp int
	//sl := []string{}
	aa := r.FormValue("in")
	session, _ := store.Get(r, "session-name")
	t := fmt.Sprint(session.Values["N"])
	n, _ := strconv.Atoi(t)                // кол-во цифр в загаданном числе
	zz := fmt.Sprint(session.Values["ZZ"]) // загаданное число
	temp_popytka := fmt.Sprint(session.Values["Popytka"])
	popytka, err := strconv.Atoi(temp_popytka) // попытки
	if err != nil {
		fmt.Println("func app: Ошибка конвертации Atoi\n", err.Error())
	}
	//todo проверить, как читается срез из сессии
	var x interface{} = session.Values["Sl"] // x имеет динамический тип
	sl, ok := x.([]string)                   // sl имеет тип []string
	fmt.Println("ok перевода =", ok)
	fmt.Println(sl)

	if aa == "" || len(aa) != N {
		// не правильный ввод попытки
		t, err := template.ParseFiles("templates/game.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Game.Name = fmt.Sprint(session.Values["CurName"])
		Game.Score = scoreUserDB(Game.Name)
		Game.Popytka = popytka
		Game.Com = "1" // чтоб вывести коментарий о неправильном вводе
		Game.Sl = sl
		t.ExecuteTemplate(w, "game", Game)
	} else {
		// если правильно введена попытка
		//todo прочитать все из сессии для игры

		zag := make([]string, n)
		a := make([]string, n)
		m := make([]string, n)
		for i, s := range zz { // раскладываем загаданное число
			zag[i] = string(s)
			m[i] = string(s)
		}
		for i, s := range aa { // раскладываем введенное число
			a[i] = string(s)
		}
		// проверяем на быков
		for i := 0; i < n; i++ {
			if a[i] == zag[i] {
				m[i] = "b"
				a[i] = "B"
			}
		}
		// проверяем на коров
		for i := 0; i < n; i++ {
			temp = indexCow(m, a[i])
			if temp > -1 {
				m[temp] = "k"
			}
		}
		//todo остановился где-то здесь
		if countCow(m, "b") == n {
			// если соличество "b" == n т.е. все быки. ПОБЕДА
			fmt.Println("поздравляю! Вы угадали с ", Popytka, " попытки ")
			stemp := fmt.Sprint(aa, " - ", countCow(m, "b"), " БЫКОВ  ", countCow(m, "k"), " КОРОВ")
			fmt.Println(stemp)
			Sl = append(Sl, stemp)

			// подсчет очков
			scor := 1
			for i := 0; i < n; i++ {
				scor = scor * 10
			}
			maxPopyt := n * 10
			scor = scor * (maxPopyt - Popytka + 1)
			fmt.Println(scor, "очков")
			newScore := addScoreDB(CurName, scor)
			fmt.Println("ВСЕГО: ", newScore, "ОЧКОВ")

			t, err := template.ParseFiles("templates/game.html",
				"templates/header.html", "templates/footer.html")
			if err != nil {
				fmt.Fprintf(w, err.Error())
			}
			Game.Name = CurName
			Game.Score = newScore
			Game.Popytka = Popytka
			Game.Com = "2"
			Game.Sl = Sl
			t.ExecuteTemplate(w, "game", Game)

		} else {
			// если не победа
			stemp := fmt.Sprint(aa, " - ", countCow(m, "b"), " БЫКОВ  ", countCow(m, "k"), " КОРОВ")
			fmt.Println(stemp)
			sl = append(sl, stemp)
			popytka++

			for i := 0; i < n; i++ {
				m[i] = Zag[i]
			}

			t, err := template.ParseFiles("templates/game.html",
				"templates/header.html", "templates/footer.html")
			if err != nil {
				fmt.Fprintf(w, err.Error())
			}
			Game.Name = CurName
			Game.Score = scoreUserDB(CurName)
			Game.Popytka = Popytka
			Game.Com = "0"
			Game.Sl = Sl

			t.ExecuteTemplate(w, "game", Game)
		}
	}
}

func handleFunc() {
	http.HandleFunc("/", index)
	http.HandleFunc("/newgame", newgame)
	http.HandleFunc("/verif_user", verif_user)
	http.HandleFunc("/new_user", new_user)
	http.HandleFunc("/new_game", new_game)
	http.HandleFunc("/add_new_user", addNewUser)
	http.HandleFunc("/game_app", app)

	http.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static/"))))

	http.ListenAndServe(":5000", nil)
}

func main() {
	createTableDB()
	handleFunc()
}
