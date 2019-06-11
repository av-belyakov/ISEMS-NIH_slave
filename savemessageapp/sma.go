package savemessageapp

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//PathDirLocationLogFiles место расположения лог-файлов приложения
type PathDirLocationLogFiles struct {
	pathLogFiles string
}

//pathConfig путь до директории с лог-файлами
type pathConfig struct {
	PathLogFiles string `json:"pathLogFiles"`
}

//New конструктор для огранизации записи лог-файлов
func New() *PathDirLocationLogFiles {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	var pc pathConfig

	//читаем основной конфигурационный файл в формате JSON
	err = readMainConfig(dir+"/config.json", &pc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var pathDirLocationLogFiles PathDirLocationLogFiles
	pathDirLocationLogFiles.pathLogFiles = pc.PathLogFiles

	return &pathDirLocationLogFiles
}

//readMainConfig читает основной конфигурационный файл и сохраняет в pathConfig
func readMainConfig(fileName string, pc *pathConfig) error {
	var err error
	row, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(row, &pc)
	if err != nil {
		return err
	}

	return err
}

func createLogsDirectory(pathLogFiles, directoryName string) error {
	files, err := ioutil.ReadDir(pathLogFiles)
	if err != nil {
		return err
	}

	for _, fl := range files {
		if fl.Name() == directoryName {
			return nil
		}
	}

	err = os.Mkdir(pathLogFiles+"/"+directoryName, 0777)
	if err != nil {
		return err
	}

	return nil
}

func compressLogFile(filePath string, fileName string, fileSize int64) error {
	fd, err := os.Open(filePath + fileName)
	if err != nil {
		return err
	}
	defer fd.Close()

	fileInfo, err := fd.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() < fileSize {
		return nil
	}

	timeNowUnix := time.Now().Unix()
	newFileName := strconv.FormatInt(timeNowUnix, 10) + "_" + strings.Replace(fileName, ".log", ".gz", -1)

	fileIn, err := os.Create(filePath + newFileName)
	if err != nil {
		return err
	}
	defer fileIn.Close()

	zw := gzip.NewWriter(fileIn)
	zw.Name = newFileName

	fileOut, err := ioutil.ReadFile(filePath + fileName)
	if err != nil {
		return err
	}

	if _, err := zw.Write(fileOut); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	_ = os.Remove((filePath + fileName))

	return nil
}

//LogMessage сохраняет в лог файлах сообщения об ошибках или информационные сообщения
func (pdllf *PathDirLocationLogFiles) LogMessage(typeMessage, message string) (err error) {
	const logDirName = "isems-nih_slave_logs"
	const logFileSize = 50000000

	fileNameTypeMessage := map[string]string{
		"error": "error_slave.log",
		"info":  "info_slave.log",
	}

	if typeMessage == "" || message == "" {
		return errors.New("'typeMessage' or 'message' empty variable")
	}

	if err = createLogsDirectory(pdllf.pathLogFiles, logDirName); err != nil {
		return err
	}

	_ = compressLogFile(pdllf.pathLogFiles+"/"+logDirName+"/", fileNameTypeMessage[typeMessage], logFileSize)

	var fileOut *os.File
	fileOut, err = os.OpenFile(pdllf.pathLogFiles+"/"+logDirName+"/"+fileNameTypeMessage[typeMessage], os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err)

		return err
	}

	timeNowString := time.Now().String()
	tns := strings.Split(timeNowString, " ")

	writer := bufio.NewWriter(fileOut)
	defer func() {
		if err == nil {
			err = writer.Flush()
		}
	}()

	strMsg := fmt.Sprintf("%s %s [%s %s] - %s\n", tns[0], tns[1], tns[2], tns[3], message)

	if _, err = writer.WriteString(strMsg); err != nil {
		return err
	}

	return err
}
