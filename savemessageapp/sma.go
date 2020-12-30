package savemessageapp

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//PathDirLocationLogFiles место расположения лог-файлов приложения
type PathDirLocationLogFiles struct {
	pathLogFiles     string
	fileNameType     map[string]string
	fileDescriptor   map[string]*os.File
	logDirName       string
	logFileSize      int64
	chanWriteMessage chan chanReqSettings
}

//chanReqSettings канал для записи информационных сообщений в лог файлы
type chanReqSettings struct {
	typeLogMessage *TypeLogMessage
}

//configPathConfig путь до директории с лог-файлами
type configPath struct {
	PathLogFiles string `json:"pathLogFiles"`
}

//TypeLogMessage описание типа для записи логов
// TypeMessage - тип "error", "info", если пусто значит "error"
// Description - описание
// FuncName - имя функции
type TypeLogMessage struct {
	TypeMessage, Description, FuncName string
}

//New конструктор для огранизации записи лог-файлов
func New() (*PathDirLocationLogFiles, error) {
	var cp configPath
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	//читаем основной конфигурационный файл в формате JSON
	err = readMainConfig(path.Join(dir, "config.json"), &cp)
	if err != nil {
		log.Fatal(err)
	}

	pdllf := PathDirLocationLogFiles{
		pathLogFiles: cp.PathLogFiles,
		fileNameType: map[string]string{
			"error": "error_message.log",
			"info":  "info_message.log",
		},
		fileDescriptor:   make(map[string]*os.File),
		logDirName:       "isems-nih_slave_logs",
		logFileSize:      5000000,
		chanWriteMessage: make(chan chanReqSettings),
	}

	if err = createLogsDirectory(pdllf.pathLogFiles, pdllf.logDirName); err != nil {
		return &pdllf, err
	}

	for n := range pdllf.fileNameType {
		fd, err := os.OpenFile(path.Join(pdllf.pathLogFiles, pdllf.logDirName, pdllf.fileNameType[n]), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return &pdllf, err
		}
		pdllf.fileDescriptor[n] = fd
	}

	go func() {
		defer func() {
			for n := range pdllf.fileDescriptor {
				pdllf.fileDescriptor[n].Close()
			}
		}()

		for msg := range pdllf.chanWriteMessage {
			pdllf.writeMessgae(msg.typeLogMessage)

			fi, _ := pdllf.fileDescriptor[msg.typeLogMessage.TypeMessage].Stat()
			if fi.Size() > pdllf.logFileSize {
				pdllf.compressFile(msg.typeLogMessage.TypeMessage)

				pdllf.fileDescriptor[msg.typeLogMessage.TypeMessage].Close()
				delete(pdllf.fileDescriptor, msg.typeLogMessage.TypeMessage)
				_ = os.Remove(path.Join(pdllf.pathLogFiles, pdllf.logDirName, pdllf.fileNameType[msg.typeLogMessage.TypeMessage]))

				fd, _ := os.OpenFile(path.Join(pdllf.pathLogFiles, pdllf.logDirName, pdllf.fileNameType[msg.typeLogMessage.TypeMessage]), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
				pdllf.fileDescriptor[msg.typeLogMessage.TypeMessage] = fd
			}
		}
	}()

	return &pdllf, nil
}

//LogMessage сохраняет в лог файлах сообщения об ошибках или информационные сообщения
func (pdllf *PathDirLocationLogFiles) LogMessage(tlm TypeLogMessage) {
	typeMessage := tlm.TypeMessage
	if typeMessage == "" && tlm.Description == "" {
		return
	}

	if typeMessage == "" {
		typeMessage = "error"
	}

	funcName := ""
	if tlm.FuncName != "" {
		funcName = fmt.Sprintf(" (function '%s')", tlm.FuncName)
	}

	pdllf.chanWriteMessage <- chanReqSettings{
		typeLogMessage: &TypeLogMessage{
			FuncName:    funcName,
			TypeMessage: typeMessage,
			Description: tlm.Description,
		},
	}
}

func (pdllf *PathDirLocationLogFiles) writeMessgae(tlm *TypeLogMessage) {
	var err error

	timeNowString := time.Now().String()
	tns := strings.Split(timeNowString, " ")
	strMsg := fmt.Sprintf("%s %s [%s %s] - %s%s\n", tns[0], tns[1], tns[2], tns[3], tlm.Description, tlm.FuncName)

	fd := pdllf.fileDescriptor[tlm.TypeMessage]
	writer := bufio.NewWriter(fd)
	defer func() {
		if err == nil {
			err = writer.Flush()
		}
	}()

	if _, err = writer.WriteString(strMsg); err != nil {
		log.Printf("func 'LogMessage' ERROR: '%v'\n", err)
	}
}

func (pdllf *PathDirLocationLogFiles) compressFile(tm string) {
	timeNowUnix := time.Now().Unix()
	fn := strconv.FormatInt(timeNowUnix, 10) + "_" + strings.Replace(pdllf.fileNameType[tm], ".log", ".gz", -1)

	fileIn, err := os.Create(path.Join(pdllf.pathLogFiles, pdllf.logDirName, fn))
	if err != nil {
		return
	}
	defer fileIn.Close()

	zw := gzip.NewWriter(fileIn)
	zw.Name = fn

	fileOut, err := ioutil.ReadFile(path.Join(pdllf.pathLogFiles, pdllf.logDirName, pdllf.fileNameType[tm]))
	if err != nil {
		return
	}

	if _, err := zw.Write(fileOut); err != nil {
		return
	}

	_ = zw.Close()
}

//ReadMainConfig читает основной конфигурационный файл и сохраняет данные
func readMainConfig(fileName string, cp *configPath) error {
	var err error
	row, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(row, &cp)
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

	err = os.Mkdir(path.Join(pathLogFiles, directoryName), 0777)
	if err != nil {
		return err
	}

	return nil
}
