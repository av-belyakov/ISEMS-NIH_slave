package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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

/*var acc configure.AccessClientsConfigure
var listAccessIPAddress ListAccessIPAddress

//здесь хранится информация о всех задачах по фильтрации
var ift configure.InformationFilteringTask

//здесь хранится информация о всех задачах по выгрузке сет. трафика
var dfi configure.DownloadFilesInformation
*/

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
	pattern := `^Application\sISEMS-NIH\s(master|slave),\sv\d+\.\d+`
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

	//получаем номер версии приложения
	if err = getVersionApp(&appConfig); err != nil {
		_ = saveMessageApp.LogMessage("error", "it is impossible to obtain the version number of the application")
	}

}

func main() {
	smta := configure.NewRepositorySMTA()

	modulenetworkinteraction.MainNetworkInteraction(&appConfig, smta)
}
