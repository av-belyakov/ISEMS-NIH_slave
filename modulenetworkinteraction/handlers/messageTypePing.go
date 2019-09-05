package handlers

/*
* Обработчик сообщений типа 'Ping'
*
* Версия 0.2, дата релиза 30.05.2019
* */

import (
	"encoding/json"
	"fmt"
	"strings"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulefiltrationfile"
	"ISEMS-NIH_slave/savemessageapp"
)

type checkingTaskResult struct {
	isComplite bool
	Error      error
}

//HandlerMessageTypePing обработчик сообщений типа 'Ping'
func HandlerMessageTypePing(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission) {

	reqJSON := configure.MsgTypePing{}
	if err := json.Unmarshal(*req, &reqJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	var mcpf int8 = 3
	if mcpf > 0 {
		mcpf = reqJSON.Info.MaxCountProcessFiltration
	}

	typeAreaNetwork := "ip"
	if strings.ToLower(reqJSON.Info.TypeAreaNetwork) == "pppoe" {
		typeAreaNetwork = "pppoe"
	}

	sma.SetApplicationSetting(configure.ApplicationSettings{
		TypeAreaNetwork: typeAreaNetwork,
		StorageFolders:  reqJSON.Info.StorageFolders,
	})

	cs, ok := sma.GetClientSetting(clientID)
	if !ok {
		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("unable to send message of type 'pong' client with ID %v does not exist", clientID))

		return
	}

	cs.MaxCountProcessFiltration = mcpf
	cs.SendsTelemetry = reqJSON.Info.EnableTelemetry

	sma.SetClientSetting(clientID, cs)

	resJSON, err := json.Marshal(configure.MsgTypePing{
		MsgType: "pong",
		Info:    configure.DetailInfoMsgPing{},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	cwtResText <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &resJSON,
	}

	chanCheckTask := checkingExecuteTaskFiltration(cwtResText, sma, clientID)

	for r := range chanCheckTask {
		if r.isComplite {
			break
		}

		if r.Error != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(r.Error))
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
				isComplite: true,
			}
		}

		if len(taskList) == 0 {
			c <- checkingTaskResult{
				isComplite: true,
			}
		}

		for taskID, info := range taskList {
			if info.Status == "stop" || info.Status == "complite" {
				//отправляем сообщение о завершении фильтрации и передаем СПИСОК ВСЕХ найденных в результате фильтрации файлов
				if err := modulefiltrationfile.SendMessageFiltrationComplete(cwtResText, sma, clientID, taskID); err != nil {
					c <- checkingTaskResult{
						Error: err,
					}
				}
			}
		}

		c <- checkingTaskResult{
			isComplite: true,
		}
	}()

	return c
}
