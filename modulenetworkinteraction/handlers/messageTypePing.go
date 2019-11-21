package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulefiltrationfile"
	"ISEMS-NIH_slave/savemessageapp"
)

type checkingTaskResult struct {
	isComplete bool
	Error      error
}

//HandlerMessageTypePing обработчик сообщений типа 'Ping'
func HandlerMessageTypePing(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission) {

	fn := "HandlerMessageTypePing"

	reqJSON := configure.MsgTypePing{}
	if err := json.Unmarshal(*req, &reqJSON); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	typeAreaNetwork := "ip"
	if strings.ToLower(reqJSON.Info.TypeAreaNetwork) == "pppoe" {
		typeAreaNetwork = "pppoe"
	}

	sma.SetApplicationSetting(configure.ApplicationSettings{
		TypeAreaNetwork: typeAreaNetwork,
		StorageFolders:  reqJSON.Info.StorageFolders,
	})

	cs, err := sma.GetClientSetting(clientID)
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("unable to send message of type 'pong' client with ID %v does not exist", clientID),
			FuncName:    fn,
		})

		return
	}

	cs.SendsTelemetry = reqJSON.Info.EnableTelemetry

	sma.SetClientSetting(clientID, cs)

	resJSON, err := json.Marshal(configure.MsgTypePing{
		MsgType: "pong",
		Info:    configure.DetailInfoMsgPing{},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	cwtResText <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &resJSON,
	}

	chanCheckTask := checkingExecuteTaskFiltration(cwtResText, sma, clientID)

	for r := range chanCheckTask {
		if r.isComplete {
			break
		}

		if r.Error != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(r.Error),
				FuncName:    fn,
			})
		}
	}
}

//checkingExecuteTaskFiltration проверка наличие выполняемых задач и отправка информации по ним
func checkingExecuteTaskFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID string) chan checkingTaskResult {

	c := make(chan checkingTaskResult)

	go func() {
		//получаем все выполняемые данным пользователем задачи
		taskList, ok := sma.GetListTasksFiltration(clientID)
		if !ok {
			c <- checkingTaskResult{
				isComplete: true,
			}
		}

		if len(taskList) == 0 {
			c <- checkingTaskResult{
				isComplete: true,
			}
		}

		for taskID, info := range taskList {
			if info.Status == "stop" || info.Status == "complete" {
				//отправляем сообщение о завершении фильтрации и передаем СПИСОК ВСЕХ найденных в результате фильтрации файлов
				if err := modulefiltrationfile.SendMessageFiltrationComplete(cwtResText, sma, clientID, taskID); err != nil {
					c <- checkingTaskResult{
						Error: err,
					}
				}
			}
		}

		c <- checkingTaskResult{
			isComplete: true,
		}
	}()

	return c
}
