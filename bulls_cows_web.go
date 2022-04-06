package main

import (
	crypto "crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io/ioutil"
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

type GameType struct {
	// для передачи в форму game
	Name    string
	Score   int
	Popytka int
	Sl      []string // срез для вывода резудьтатов
	Com     string   // для вывода коментария в шаблон
}

type Config struct {
	MaxN     int
	PasAdmin string
}

var ConfigGame Config
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
var users []Users
var Player Users
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
	record, err := db.Query("SELECT * FROM users ORDER BY score DESC")
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
}

func verif_user(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	inPas := r.FormValue("password")
	fmt.Println("введено имя:", name, ", пароль: ", inPas)

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
		fmt.Println(name, "успешно залогинен")
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
}

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
		curName := name
		t, err := template.ParseFiles("templates/kolcif.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}

		session, _ := store.Get(r, "session-name")
		session.Values["CurName"] = curName
		session.Values["ZZ"] = ""         // загаданное число в string
		session.Values["N"] = 0           // кол-во цифр в загаданном числе
		session.Values["Sl"] = []string{} // обнуляем срез попыток
		session.Values["Popytka"] = 0
		err = session.Save(r, w)
		if err != nil {
			fmt.Fprintln(w, "ошибка обновления сессии в func addNewUser", err.Error())
			return
		}

		Player = Users{}
		Player.Name = curName
		Player.Score = scoreUserDB(curName)
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
}

func new_game(w http.ResponseWriter, r *http.Request) {
	kol := r.FormValue("kol")
	session, _ := store.Get(r, "session-name")
	tempName, _ := session.Values["CurName"]
	if tempName == nil {
		fmt.Println("странно, что пользователь не авторизован!")
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
		return
	}
	curName := fmt.Sprint(tempName)
	fmt.Println(curName+": введено цифр для начала игры: ", kol)
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
		Player.Name = curName
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
}

func app(w http.ResponseWriter, r *http.Request) {
	// ЭТО ГЛАВНАЯ ФУНКЦИЯ С ЛОГИКОЙ ИГРЫ !
	var temp int
	aa := r.FormValue("in")
	session, _ := store.Get(r, "session-name")
	curName := fmt.Sprint(session.Values["CurName"])
	tmp := fmt.Sprint(session.Values["N"])
	n, _ := strconv.Atoi(tmp)              // кол-во цифр в загаданном числе
	zz := fmt.Sprint(session.Values["ZZ"]) // загаданное число
	temp_popytka := fmt.Sprint(session.Values["Popytka"])
	popytka, err := strconv.Atoi(temp_popytka) // попытки
	if err != nil {
		fmt.Println("func app: Ошибка конвертации Atoi\n", err.Error())
	}
	var x interface{} = session.Values["Sl"] // x имеет динамический тип
	sl, _ := x.([]string)                    // sl имеет тип []string
	if aa == "" || len(aa) != n {
		// не правильный ввод попытки
		t, err := template.ParseFiles("templates/game.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		Game.Name = curName
		Game.Score = scoreUserDB(Game.Name)
		Game.Popytka = popytka
		Game.Com = "1" // чтоб вывести коментарий о неправильном вводе
		Game.Sl = sl
		t.ExecuteTemplate(w, "game", Game)
	} else {
		// если правильно введена попытка
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
		if countCow(m, "b") == n {
			// если соличество "b" == n т.е. все быки. ПОБЕДА
			fmt.Println(curName+": поздравляю! Вы угадали с ", popytka, " попытки ")
			stemp := fmt.Sprint(aa, " - ", countCow(m, "b"), " БЫКОВ  ", countCow(m, "k"), " КОРОВ")
			fmt.Println(stemp)
			sl = append(sl, stemp)

			// подсчет очков
			scor := 1
			for i := 0; i < n; i++ {
				scor = scor * 10
			}
			maxPopyt := n * 10
			scor = scor * (maxPopyt - popytka + 1)
			fmt.Println(scor, "очков")
			newScore := addScoreDB(curName, scor)
			fmt.Println("ВСЕГО: ", newScore, "ОЧКОВ")

			t, err := template.ParseFiles("templates/game.html",
				"templates/header.html", "templates/footer.html")
			if err != nil {
				fmt.Fprintf(w, err.Error())
			}
			Game.Name = curName
			Game.Score = newScore
			Game.Popytka = popytka
			Game.Com = "2"
			Game.Sl = sl
			t.ExecuteTemplate(w, "game", Game)

		} else {
			// если не победа
			stemp := fmt.Sprint(aa, " - ", countCow(m, "b"), " БЫКОВ  ", countCow(m, "k"), " КОРОВ")
			fmt.Println(curName + "(" + zz + "): " + stemp)
			sl = append(sl, stemp)
			popytka++
			session.Values["Sl"] = sl // обновляем срез попыток
			session.Values["Popytka"] = popytka
			err := session.Save(r, w)
			if err != nil {
				fmt.Fprintln(w, "ошибка обновления сессии в func app", err.Error())
				return
			}
			for i := 0; i < n; i++ {
				m[i] = zag[i]
			}

			t, err := template.ParseFiles("templates/game.html",
				"templates/header.html", "templates/footer.html")
			if err != nil {
				fmt.Fprintf(w, err.Error())
			}
			Game.Name = curName
			Game.Score = scoreUserDB(curName)
			Game.Popytka = popytka
			Game.Com = "0"
			Game.Sl = sl

			t.ExecuteTemplate(w, "game", Game)
		}
	}
}

func exit(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/exit.html",
		"templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	t.ExecuteTemplate(w, "exit", nil)
}

func adminLogin(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/adminlogin.html",
		"templates/header.html", "templates/footer.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	t.ExecuteTemplate(w, "adminlogin", "0")
}

func verifAdmin(w http.ResponseWriter, r *http.Request) {
	inPas := r.FormValue("password")
	fmt.Println("введен пароль администратора:", inPas)

	pas := ConfigGame.PasAdmin
	if inPas != pas || inPas == "" {
		fmt.Println("не совпал пароль: ", inPas, " и ", pas)
		t, err := template.ParseFiles("templates/adminlogin.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		t.ExecuteTemplate(w, "adminlogin", "1")
	} else {
		// АВТОРИЗУЕМ АДМИНИСТРАТОРА
		session, _ := store.Get(r, "session-name")
		session.Values["CurName"] = "admin"
		err := session.Save(r, w)
		if err != nil {
			fmt.Fprintln(w, "ошибка записи в сессию в func verifAdmin", err.Error())
			return
		}
		fmt.Println("Администратор успешно залогинен")
		defer admin(w, r)
	}
}

func adminEdit(w http.ResponseWriter, r *http.Request) {
	var user Users
	user.Id, _ = strconv.Atoi(r.FormValue("id"))
	user.Name = r.FormValue("name")
	user.Pas = r.FormValue("pas")
	user.Score, _ = strconv.Atoi(r.FormValue("score"))
	fmt.Println(user)

	db, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `UPDATE users SET name = ?, pas = ?, score = ? WHERE id LIKE ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(user.Name, user.Pas, user.Score, user.Id)
	if err != nil {
		log.Fatal(err)
	}

	defer admin(w, r)
}

func admin(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if fmt.Sprint(session.Values["CurName"]) != "admin" {
		// если админ не залогинен в сессии
		t, err := template.ParseFiles("templates/adminlogin.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		t.ExecuteTemplate(w, "adminlogin", "0")
	} else {
		// если админ залогинен в сессии

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
			users = append(users, user)
		}

		t, err := template.ParseFiles("templates/admin.html",
			"templates/header.html", "templates/footer.html")
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}
		t.ExecuteTemplate(w, "admin", users)
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
	http.HandleFunc("/admin_login", adminLogin)
	http.HandleFunc("/verif_admin", verifAdmin)
	http.HandleFunc("/admin", admin)
	http.HandleFunc("/admin_edit", adminEdit)
	http.HandleFunc("/exit", exit)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Println(http.ListenAndServe(":5001", nil))
}

func newConfig() {
	ConfigGame.MaxN = 10
	ConfigGame.PasAdmin = "2666"
	data, err := json.Marshal(ConfigGame)
	err = ioutil.WriteFile("config.json", data, 0600)
	if err != nil {
		panic(err)
	}
}

func main() {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		if err.Error() == "open config.json: The system cannot find the file specified." {
			fmt.Println("файл конфигурации не найден, создается новый.")
			newConfig()
		} else {
			// если ошибка открытия, но не отсутствие файла
			fmt.Println(err.Error())
			return
		}
	} else {
		err = json.Unmarshal(data, &ConfigGame)
		if err != nil {
			fmt.Println("файл конфигурации поврежден! Создан новый. \n", err)
			newConfig()
		}
	}
	fmt.Println("Пароль администратора: ", ConfigGame.PasAdmin)

	createTableDB()
	handleFunc()
}
