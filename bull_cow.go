/*
Для консоли!
введите "end" для завершения
*/

package main

import (
	crypto "crypto/rand"
	"fmt"
	"math/big"
)

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
	var n, temp int
	var aa string
	const max_n = 10 // максимальное количество знаков

	fmt.Print("введите количество цифр : ")
	for {
		_, err := fmt.Scan(&n)
		if err != nil || n <= 0 || n > max_n {
			fmt.Print("введите число от 1 до ", max_n, ": ")
		} else {
			break
		}
	}
	//TODO сделать функцию для склонения слова "цифр" для количества n
	zz := RandN(n)           // ЗАГАДОННОЕ ЧИСЛО (string)
	a := make([]string, n)   // срез для ввода попыток
	zag := make([]string, n) // срез для загаданного числа
	m := make([]string, n)
	for i, s := range zz {
		zag[i] = string(s)
		m[i] = string(s)
	}
	fmt.Println("загадано число : ", zz)
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
		//fmt.Println("m=", m, "  a=", a)

		// проверяем на коров
		for i := 0; i < n; i++ {
			temp = indexCow(m, a[i])
			if temp > -1 {
				m[temp] = "k"
			}
		}
		//fmt.Println("m=", m)

		if countCow(m, "b") == n {
			fmt.Println("поздравляю! Вы угадали с ", popytka, " попытки")
			aa = "end"
		} else {
			fmt.Println(countCow(m, "b"), " БЫКОВ  ", countCow(m, "k"), " КОРОВ")
			popytka++
			for i := 0; i < n; i++ {
				m[i] = zag[i]
			}
		}
	}

}
