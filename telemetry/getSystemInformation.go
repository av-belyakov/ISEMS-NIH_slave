package telemetry

/*
* Модуль формирования системной информации
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
	sma *configure.StoreMemoryApplication,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	fn := "GetSystemInformation"

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
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	//загрузка ЦПУ
	if err := sysInfo.CreateLoadCPU(); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	//свободное дисковое пространство
	if err := sysInfo.CreateDiskSpace(); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	numGoProg := 2

DONE:
	for {
		select {
		case errMsg := <-chanErrMsg:
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(errMsg),
				FuncName:    fn,
			})

		case <-done:
			numGoProg--

			if numGoProg == 0 {
				break DONE
			}
		}
	}

	sysInfo.Info.CurrentDateTime = time.Now().Unix() * 1000
	sysInfo.MessageType = "telemetry"

	resjson, err := json.Marshal(sysInfo)
	if err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	//рассылаем всем клиентам ожидающим телеметрию
	for _, clientID := range cl {
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resjson,
		}
	}
}
