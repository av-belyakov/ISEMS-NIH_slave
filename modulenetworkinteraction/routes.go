package modulenetworkinteraction

/*
* Маршрутизация запросов поступающих через протокол websocket
*
* Версия 0.1, дата релиза 02.04.2019
* */

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulenetworkinteraction/handlers"
	"ISEMS-NIH_slave/savemessageapp"
)

type msgTypeJSON struct {
	MsgType string `json:"messageType"`
}

//RouteWssConnect маршрутизация запросов
func RouteWssConnect(
	cwtResText chan<- configure.MsgWsTransmission,
	cwtResBinary chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	sma *configure.StoreMemoryApplication,
	cwtReq <-chan configure.MsgWsTransmission) {

	fmt.Println("START function 'RouteWssConnect'...")

	/*
					!!!!!!!!
		Пример функции для отправки уведомления об окончании фильтрации после возобновления соединения
		!!!!!!!

					func sendFilterTaskInfoAfterPingMessage(remoteIP, ExternalIP string, acc *configure.AccessClientsConfigure, ift *configure.InformationFilteringTask) {
			//инициализируем функцию конструктор для записи лог-файлов
			saveMessageApp := savemessageapp.New()

			for taskIndex, task := range ift.TaskID {
				if task.RemoteIP == remoteIP {
					if _, ok := acc.Addresses[task.RemoteIP]; ok {
						switch task.TypeProcessing {
						case "execute":
							mtfeou := configure.MessageTypeFilteringExecutedOrUnexecuted{
								MessageType: "filtering",
								Info: configure.MessageTypeFilteringExecuteOrUnexecuteInfo{
									configure.FilterInfoPattern{
										IPAddress:  task.RemoteIP,
										TaskIndex:  taskIndex,
										Processing: task.TypeProcessing,
									},
									configure.FilterCountPattern{
										CountFilesProcessed:   task.CountFilesProcessed,
										CountFilesUnprocessed: task.CountFilesUnprocessed,
										CountCycleComplete:    task.CountCycleComplete,
										CountFilesFound:       task.CountFilesFound,
										CountFoundFilesSize:   task.CountFoundFilesSize,
									},
									configure.InfoProcessingFile{
										FileName:          task.ProcessingFileName,
										DirectoryLocation: task.DirectoryFiltering,
										StatusProcessed:   task.StatusProcessedFile,
									},
								},
							}

							formatJSON, err := json.Marshal(&mtfeou)
							if err != nil {
								_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
							}

							if _, ok := acc.Addresses[task.RemoteIP]; ok {
								acc.ChanWebsocketTranssmition <- formatJSON
							}
						case "complete":
							processingwebsocketrequest.SendMsgFilteringComplite(acc, ift, taskIndex, task)
						}
					}
				}
			}
		}
	*/

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	var mtJSON msgTypeJSON

	for msg := range cwtReq {
		//		fmt.Printf("Client ID:%v, request data count: %v\n", msg.ClientID, msg.Data)

		if err := json.Unmarshal(*msg.Data, &mtJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		switch mtJSON.MsgType {
		case "ping":
			fmt.Println("--- resived message JSON 'PING', func 'RouteWssConnect'")
			fmt.Printf("client ID %v\n", msg.ClientID)

			go handlers.HandlerMessageTypePing(cwtResText, sma, msg.Data, msg.ClientID)

			//отправляем сообщение о выполняемой или выполненной задачи по фильтрации (выполняется при повторном установлении соединения)
			//						sendFilterTaskInfoAfterPingMessage(remoteIP, mc.ExternalIPAddress, acc, ift)

		case "filtration":
			fmt.Println("--- resived message JSON 'FILTRATION', func 'RouteWssConnect'")
			fmt.Printf("client ID %v\n", msg.ClientID)

			handlers.HandlerMessageTypeFiltration(cwtResText, sma, msg.Data, msg.ClientID, appc.DirectoryStoringProcessedFiles.Raw)

		case "download files":
			fmt.Println("--- resived message JSON 'DOWNLOAD FILES', func 'RouteWssConnect'")
			fmt.Printf("client ID %v\n", msg.ClientID)

		}
	}
}
