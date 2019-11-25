package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulenetworkinteraction"
	"ISEMS-NIH_slave/savemessageapp"
)

//ListAccessIPAddress хранит разрешенные для соединения ip адреса
type ListAccessIPAddress struct {
	IPAddress []string
}

//SettingsLocalServerHTTPS параметры необходимые при работе приложения в режиме сервера
type SettingsLocalServerHTTPS struct {
	IP, Port, Token string
}

var appConfig configure.AppConfig

//ReadConfig читает конфигурационный файл и сохраняет данные в appConfig
func readConfigApp(fileName string, appc *configure.AppConfig) error {
	var err error
	row, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}

	err = json.Unmarshal(row, &appc)
	if err != nil {
		return err
	}

	return err
}

//getVersionApp получает версию приложения из файла README.md
func getVersionApp(appc *configure.AppConfig) error {
	failureMessage := "version not found"
	content, err := ioutil.ReadFile(appc.RootDir + "README.md")
	if err != nil {
		return err
	}

	//Application ISEMS-NIH master, v0.1
	pattern := `^Application\sISEMS-NIH\s(master|slave),\sv\d+\.\d+\.\d+`
	rx := regexp.MustCompile(pattern)
	numVersion := rx.FindString(string(content))

	if len(numVersion) == 0 {
		appc.VersionApp = failureMessage
		return nil
	}

	s := strings.Split(numVersion, " ")
	if len(s) < 3 {
		appc.VersionApp = failureMessage
		return nil
	}

	appc.VersionApp = s[3]

	return nil
}

func createStoreDirectory(dirPath string) error {
	mkDirectory := func(rootDir, createDir string) error {
		if rootDir == "" {
			return nil
		}

		files, err := ioutil.ReadDir(rootDir)
		if err != nil {
			return err
		}

		for _, fl := range files {
			if fl.Name() == createDir {
				return nil
			}
		}

		pathDir := path.Join(rootDir, createDir)
		if rootDir == "/" {
			pathDir = fmt.Sprintf("%v", createDir)
		}

		err = os.Mkdir(pathDir, 0777)
		if err != nil {
			return err
		}

		return nil
	}

	if strings.Count(dirPath, "/") == 1 {
		if err := mkDirectory("/", dirPath); err != nil {
			return err
		}

		return nil
	}

	list := strings.Split(dirPath, "/")
	for i := 0; i < len(list)-1; i++ {
		rd := list[i]
		if i != 0 {
			rd = strings.Join(list[:i+1], "/")
		}

		if err := mkDirectory(rd, list[i+1]); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	var err error

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	//проверяем наличие tcpdump
	func() {
		stdout, err := exec.Command("sh", "-c", "whereis tcpdump").Output()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		list := strings.Split(string(stdout), " ")

		if !strings.Contains(list[1], "tcpdump") {
			fmt.Println("tcpdump is not found")
			os.Exit(1)
		}
	}()

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	//читаем конфигурационный файл приложения
	err = readConfigApp(dir+"/config.json", &appConfig)
	if err != nil {
		fmt.Println("Error reading configuration file", err)
		os.Exit(1)
	}

	appConfig.RootDir = dir + "/"

	//для сервера обеспечивающего подключение
	appConfig.LocalServerHTTPS.PathCertFile = appConfig.RootDir + appConfig.LocalServerHTTPS.PathCertFile
	appConfig.LocalServerHTTPS.PathPrivateKeyFile = appConfig.RootDir + appConfig.LocalServerHTTPS.PathPrivateKeyFile

	//создаем основную директорию куда будут сохраняться обработанные при выполнении фильтрации файлы
	if err = createStoreDirectory(appConfig.DirectoryStoringProcessedFiles.Raw); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{Description: fmt.Sprint(err)})
	}

	//создаем основную директорию куда будут сохраняться обработанные файлы (при выделении объектов)
	if err = createStoreDirectory(appConfig.DirectoryStoringProcessedFiles.Object); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{Description: fmt.Sprint(err)})
	}

	//получаем номер версии приложения
	if err = getVersionApp(&appConfig); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: "it is impossible to obtain the version number of the application",
		})
	}

	//проверяем размер передаваемой части файла
	if (appConfig.MaxSizeTransferredChunkFile < 1024) || (appConfig.MaxSizeTransferredChunkFile > 65535) {
		appConfig.MaxSizeTransferredChunkFile = 1024
	}

}

func main() {
	defer func() {
		if err := recover(); err != nil {
			saveMessageApp := savemessageapp.New()
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				TypeMessage: "error",
				Description: fmt.Sprintf("STOP 'main' function, Error:'%v'", err),
				FuncName:    "main",
			})
		}
	}()

	//создаем новый репозиторий для хранения информации, в том числе о задачах
	sma := configure.NewRepositorySMA()

	log.Printf("START application ISEMS-NIH_slave version %q\n", appConfig.VersionApp)

	modulenetworkinteraction.MainNetworkInteraction(&appConfig, sma)
}
