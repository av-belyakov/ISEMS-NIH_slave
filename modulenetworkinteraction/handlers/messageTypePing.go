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
	err        error
}

//HandlerMessageTypePing обработчик сообщений типа 'Ping'
func HandlerMessageTypePing(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string,
	appc *configure.AppConfig,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission) {

	fn := "HandlerMessageTypePing"

	reqJSON := configure.MsgTypePing{}
	if err := json.Unmarshal(*req, &reqJSON); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
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
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("unable to send message of type 'pong' client with ID %v does not exist", clientID),
			FuncName:    fn,
		})

		return
	}

	cs.SendsTelemetry = reqJSON.Info.EnableTelemetry

	sma.SetClientSetting(clientID, cs)

	resJSON, err := json.Marshal(configure.MsgTypePong{
		MsgType: "pong",
		Info: configure.DetailInfoMsgPong{
			AppVersion:     appc.VersionApp,
			AppReleaseDate: appc.DateCreateApp,
		},
	})

	cwtResText <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &resJSON,
	}

	go checkingExecuteTaskFiltration(cwtResText, sma, clientID, saveMessageApp)
}

//checkingExecuteTaskFiltration проверка наличие выполняемых задач и отправка информации по ним
func checkingExecuteTaskFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {
	funcName := "checkingExecuteTaskFiltration"

	//получаем все выполняемые данным пользователем задачи
	taskList, ok := sma.GetListTasksFiltration(clientID)
	if !ok {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			TypeMessage: "info",
			Description: fmt.Sprint("func 'checkingExecuteTaskFiltration', filtration task processed not found, client id not found"),
			FuncName:    funcName,
		})

		return
	}

	if len(taskList) == 0 {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			TypeMessage: "info",
			Description: fmt.Sprint("func 'checkingExecuteTaskFiltration', filtration task processed not found, task list is empty"),
			FuncName:    funcName,
		})

		return
	}

	for taskID, info := range taskList {
		if info.Status == "stop" || info.Status == "complete" {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				TypeMessage: "info",
				Description: fmt.Sprintf("filtration task processed is 'stop' or 'complete', send message about 'complete' to isems-nih_master with id '%v'", clientID),
				FuncName:    funcName,
			})

			//отправляем сообщение о завершении фильтрации и передаем СПИСОК ВСЕХ найденных в результате фильтрации файлов
			if err := modulefiltrationfile.SendMessageFiltrationComplete(cwtResText, sma, clientID, taskID); err != nil {
				saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    funcName,
				})

				continue
			}

			//удаляем задачу
			sma.DelTaskFiltration(clientID, taskID)
		}
	}
}
