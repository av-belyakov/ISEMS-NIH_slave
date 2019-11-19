package common

import (
	"regexp"
	"strconv"
	"strings"
)

//CheckStringIP проверка ip адреса принятого в виде строки
func CheckStringIP(ip string) (bool, error) {
	pattern := "^((25[0-5]|2[0-4]\\d|[01]?\\d\\d?)[.]){3}(25[0-5]|2[0-4]\\d|[01]?\\d\\d?)$"

	rx, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	return rx.MatchString(ip), nil
}

//CheckStringNetwork проверка подсети принятой в виде строки вида 0.0.0.0/32
func CheckStringNetwork(nw string) (bool, error) {
	i := strings.Split(nw, "/")
	if len(i) <= 1 {
		return false, nil
	}

	ok, err := CheckStringIP(i[0])
	if err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	o, err := strconv.Atoi(i[1])
	if err != nil {
		return false, nil
	}

	if o == 0 || o > 32 {
		return false, nil
	}

	return true, nil
}

//CheckStringToken проверяет токен полученный от пользователя
func CheckStringToken(str string) (bool, error) {
	pattern := "^\\w+$"

	rx, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	return rx.MatchString(str), nil
}

//CheckFolders проверяет имена директорий
func CheckFolders(f []string) (bool, error) {
	if len(f) == 0 {
		return false, nil
	}

	pattern := "^(/|_|\\w)+$"
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	for _, v := range f {
		if !rx.MatchString(v) {
			return false, nil
		}
	}

	return true, nil
}
