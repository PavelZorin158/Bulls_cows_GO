/*
Для консоли! подключена БД
во время игры введите "end" для завершения
*/

package main

import (
	crypto "crypto/rand"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math/big"
)

const maxN = 10 // максимальное количество знаков

func printUsers(db *sql.DB) {
	// печать всех пользователей
	record, err := db.Query("SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	for record.Next() {
		var id int
		var name string
		var pas string
		var score int
		record.Scan(&id, &name, &pas, &score)
		fmt.Printf("User: %d %s %d\n", id, name, score)
	}

}

func createTable(db *sql.DB) {
	// создание таблицы в файле БД, если ее там нет
	users_table := `CREATE TABLE IF NOT EXISTS users (
        id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
        "name" TEXT,
        "pas" TEXT,
        "score" INTEGER  DEFAULT 0);`
	query, err := db.Prepare(users_table)
	if err != nil {
		log.Fatal(err)
	}
	query.Exec()
}

func verifUser(db *sql.DB, name string) string {
	// возвращает пароль пользователя name
	var pas string
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

func loginUser(db *sql.DB) string {
	// при удачной авторизации возвращает имя пользователя
	var name, pas, mypas string
	fmt.Print("Имя: ")
	fmt.Scan(&name)
	pas = verifUser(db, name)
	for pas == "" {
		fmt.Print("Пользователя с таким именем НЕ существует!\n" +
			"Выберите другое: ")
		fmt.Scan(&name)
		pas = verifUser(db, name)
	}
	fmt.Print("пароль: ")
	fmt.Scan(&mypas)
	for pas != mypas {
		fmt.Print("НЕ правильный пароль, попробуйте другой: ")
		fmt.Scan(&mypas)
	}
	return name
}

func scoreUser(db *sql.DB, name string) int {
	// возвращает score для пользователя name
	var scor int
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

func addScore(db *sql.DB, name string, scor int) int {
	// добавляет у пользователя name  в БД к score, scor и перезаписывает
	// возвращает итоговый score
	score := scoreUser(db, name) + scor
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

func addUser(db *sql.DB) {
	// добавление пользователя в БД
	var name, pas string
	fmt.Println("*** Новый пользователь ***")
	fmt.Print("Имя: ")
	fmt.Scan(&name)
	pas = verifUser(db, name)
	for pas != "" {
		fmt.Print("Пользователь с таким именем уже существует!\n" +
			"Выберите другое: ")
		fmt.Scan(&name)
		pas = verifUser(db, name)
	}
	fmt.Print("пароль: ")
	fmt.Scan(&pas)
	for pas == "" || len(pas) < 3 {
		fmt.Println("длинна =", len(pas), "символов")
		fmt.Print("Выберите другой пароль: ")
		fmt.Scan(&pas)
	}
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

func main() {
	var ins string
	var name string
	filedb, err := sql.Open("sqlite3", "cow.db")
	if err != nil {
		fmt.Println("Ошибка открытия COW.DB")
		fmt.Println(err.Error())
		return
	} else {
		defer filedb.Close()
	}
	createTable(filedb)

	for {
		if name != "" {
			// если пользователь авторизован (в name есть имя)
			fmt.Print("1 - продолжить, 2 - выйти : ")
			fmt.Scan(&ins)
			if ins == "2" {
				name = ""
				continue
			} else if ins == "1" {
				ins = "game"
			} else {
				continue
			}
		}
		if ins != "game" {
			printUsers(filedb)
			fmt.Print("1 - регистрация, 2 - войти, 3 - выход : ")
			fmt.Scan(&ins)
		}
		if ins == "3" {
			break
		} else if ins == "1" {
			addUser(filedb)
		} else if ins == "2" {
			name = loginUser(filedb)
			fmt.Println(name, "авторизован. результат -", scoreUser(filedb, name), "очков")
		}
		if name != "" {
			// НАЧАЛО ИГРЫ
			var n, temp int
			var aa string
			fmt.Print("введите количество цифр : ")
			for {
				_, err := fmt.Scan(&n)
				if err != nil || n <= 0 || n > maxN {
					fmt.Print("введите число от 1 до ", maxN, ": ")
				} else {
					break
				}
			}
			zz := RandN(n)           // ЗАГАДОННОЕ ЧИСЛО (string)
			a := make([]string, n)   // срез для ввода попыток
			zag := make([]string, n) // срез для загаданного числа
			m := make([]string, n)
			for i, s := range zz {
				zag[i] = string(s)
				m[i] = string(s)
			}
			// fmt.Println("загадано число : ", zz)
			popytka := 1

			for aa != "end" {
				fmt.Print("попытка ", popytka, ": ")
				_, err := fmt.Scan(&aa)
				if aa == "end" {
					continue
				}
				if err != nil || len(aa) != n {
					fmt.Println("вводите ", n, " цифр!")
					continue
				}
				for i, s := range aa {
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
				//fmt.Println("m=", m)

				if countCow(m, "b") == n {
					fmt.Print("поздравляю! Вы угадали с ", popytka, " попытки ")
					aa = "end"
					scor := 1
					for i := 0; i < n; i++ {
						scor = scor * 10
					}
					maxPopyt := n * 10
					scor = scor * (maxPopyt - popytka + 1)
					fmt.Println(scor, "очков")
					fmt.Println("ВСЕГО: ", addScore(filedb, name, scor), "ОЧКОВ")
				} else {
					fmt.Println(countCow(m, "b"), " БЫКОВ  ", countCow(m, "k"), " КОРОВ")
					popytka++
					for i := 0; i < n; i++ {
						m[i] = zag[i]
					}
				}
			}

		}
	}
}
