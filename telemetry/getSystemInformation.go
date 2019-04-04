package telemetry

/*
* Модуль формирования системной информации
*
* Версия 0.1, дата релиза 04.04.2019
* */

import (
	"encoding/json"
	"fmt"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//GetSystemInformation позволяет получить системную информацию
func GetSystemInformation(
	cwtResText chan<- configure.MsgWsTransmission,
	cl []string,
	sma *configure.StoreMemoryApplication) {

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	var sysInfo SysInfo

	chanErrMsg := make(chan error)
	done := make(chan struct{})

	as := sma.GetApplicationSetting()

	//временной интервал файлов хранящихся на дисках
	go sysInfo.CreateFilesRange(done, chanErrMsg, as.StorageFolders)

	//нагрузка на сетевых интерфейсах
	go sysInfo.CreateLoadNetworkInterface(done, chanErrMsg)

	//загрузка оперативной памяти
	if err := sysInfo.CreateRandomAccessMemory(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//загрузка ЦПУ
	if err := sysInfo.CreateLoadCPU(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//свободное дисковое пространство
	if err := sysInfo.CreateDiskSpace(); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	numGoProg := 2

DONE:
	for {
		select {
		case errMsg := <-chanErrMsg:
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(errMsg))

		case <-done:
			numGoProg--

			if numGoProg == 0 {
				break DONE
			}
		}
	}

	//получаем локальный ip
	var localAddr string
	links := sma.GetClientsListConnection()
	for _, l := range links {
		localAddr = l.Link.LocalAddr().String()
		return
	}

	sysInfo.Info.IPAddress = localAddr
	sysInfo.Info.CurrentDateTime = time.Now().Unix() * 1000
	sysInfo.MessageType = "information"

	resjson, err := json.Marshal(sysInfo)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//рассылаем всем клиентам ожидающим телеметрию
	for _, clientID := range cl {
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resjson,
		}
	}
}
