package common

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"math"
	"os"
	"strconv"
	"time"

	"ISEMS-NIH_slave/configure"
)

//GetUniqIDFormatMD5 генерирует уникальный идентификатор в формате md5
func GetUniqIDFormatMD5(str string) string {
	currentTime := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, str+"_"+strconv.FormatInt(currentTime, 10))

	hsum := hex.EncodeToString(h.Sum(nil))

	return hsum
}

//GetCountPartsMessage получить количество частей сообщений
func GetCountPartsMessage(list map[string]int, sizeChunk int) int {
	var maxFiles float64
	for _, v := range list {
		if maxFiles < float64(v) {
			maxFiles = float64(v)
		}
	}

	newCountChunk := float64(sizeChunk)
	x := math.Floor(maxFiles / newCountChunk)
	y := maxFiles / newCountChunk

	if (y - x) != 0 {
		x++
	}

	return int(x)
}

//CountNumberParts подсчет количества частей
func CountNumberParts(num int64, sizeChunk int) int {
	n := float64(num)
	sc := float64(sizeChunk)
	x := math.Floor(n / sc)
	y := n / sc

	if (y - x) != 0 {
		x++
	}

	return int(x)
}

//GetChunkListFiles разделяет список файлов на кусочки
func GetChunkListFiles(numPart, sizeChunk, countParts int, listFilesFilter map[string][]string) map[string][]string {
	lff := map[string][]string{}

	for disk, files := range listFilesFilter {
		if numPart == 1 {
			if len(files) < sizeChunk {
				lff[disk] = files[:]
			} else {
				lff[disk] = files[:sizeChunk]
			}

			continue
		}

		num := sizeChunk * (numPart - 1)
		numEnd := num + sizeChunk

		if numPart == countParts {
			if num < len(files) {
				lff[disk] = files[num:]

				continue
			}

			lff[disk] = []string{}
		}

		if numPart < countParts {
			if num > len(files) {
				lff[disk] = []string{}

				continue
			}

			if numEnd < len(files) {
				lff[disk] = files[num:numEnd]

				continue
			}

			lff[disk] = files[num:]
		}

	}

	return lff
}

//GetChunkListFilesFound делит отображение с информацией о файлах на отдельные части
func GetChunkListFilesFound(lf map[string]*configure.InputFilesInformation, numPart, countParts, sizeChunk int) map[string]*configure.InputFilesInformation {
	lnf := make([]string, 0, len(lf))
	for fname := range lf {
		lnf = append(lnf, fname)
	}

	chunk := make([]string, 0, len(lnf))
	listFiles := map[string]*configure.InputFilesInformation{}

	if numPart == 1 {
		if len(lnf) < sizeChunk {
			chunk = lnf[:]
		} else {
			chunk = lnf[:sizeChunk]
		}
	} else {

		num := sizeChunk * (numPart - 1)
		numEnd := num + sizeChunk

		if (numPart == countParts) && (num < len(lf)) {
			chunk = lnf[num:]
		}
		if (numPart < countParts) && (numEnd < len(lf)) {
			chunk = lnf[num:numEnd]
		}
	}

	for _, fname := range chunk {
		listFiles[fname] = lf[fname]
	}

	return listFiles
}

//GetFileParameters получает параметры файла, его размер и хеш-сумму
func GetFileParameters(filePath string) (int64, string, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		return 0, "", err
	}
	defer fd.Close()

	fileInfo, err := fd.Stat()
	if err != nil {
		return 0, "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, fd); err != nil {
		return 0, "", err
	}

	return fileInfo.Size(), hex.EncodeToString(h.Sum(nil)), nil
}
