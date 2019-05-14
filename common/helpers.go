package common

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"math"
	"strconv"
	"time"
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
