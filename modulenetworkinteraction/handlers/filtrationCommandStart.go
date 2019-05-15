package handlers

/*
* Модуль выполняется при получении команды фильтрации 'start'
*
* Версия 0.1, дата релиза 14.05.2019
* */

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulefiltrationfile"
	"ISEMS-NIH_slave/savemessageapp"
)

//StartFiltration обработчик команды 'start' раздела фильтрации
func StartFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	mtfcJSON *configure.MsgTypeFiltrationControl,
	clientID, directoryStoringProcessedFiles string) {

	saveMessageApp := savemessageapp.New()

	taskID := mtfcJSON.Info.TaskID
	errMsg := configure.MsgTypeError{
		MsgType: "error",
		Info: configure.DetailInfoMsgError{
			TaskID: taskID,
		},
	}

	if mtfcJSON.Info.NumberMessagesFrom[0] == 0 {
		cs, ok := sma.GetClientSetting(clientID)
		if !ok {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("client with ID %v not found", clientID))

			return
		}

		mcpf := cs.MaxCountProcessFiltration

		tasksList, _ := sma.GetListTasksFiltration(clientID)

		//проверяем количество выполняемых задач (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		if len(tasksList) >= int(mcpf) {
			errMsg.Info.ErrorName = "max limit task filtration"
			errMsg.Info.ErrorDescription = "Достигнут лимит максимального количества выполняемых параллельных задач по фильтрации файлов"

			msgJSON, err := json.Marshal(errMsg)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

				return
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &msgJSON,
			}

			return
		}

		//проверяем параметры фильтрации (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		if msg, ok := common.CheckParametersFiltration(&mtfcJSON.Info.Options); !ok {
			errMsg.Info.ErrorName = "invalid value received"
			errMsg.Info.ErrorDescription = msg

			msgJSON, err := json.Marshal(errMsg)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

				return
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &msgJSON,
			}

			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

			return
		}

		//и если параметры верны создаем новую задачу
		sma.AddTaskFiltration(clientID, taskID, &configure.FiltrationTasks{
			DateTimeStart:      mtfcJSON.Info.Options.DateTime.Start,
			DateTimeEnd:        mtfcJSON.Info.Options.DateTime.End,
			Protocol:           mtfcJSON.Info.Options.Protocol,
			UseIndex:           mtfcJSON.Info.IndexIsFound,
			CountIndexFiles:    mtfcJSON.Info.CountIndexFiles,
			Filters:            mtfcJSON.Info.Options.Filters,
			ChanStopFiltration: make(chan struct{}),
			ListFiles:          make(map[string][]string),
		})
	}

	if !mtfcJSON.Info.IndexIsFound {
		modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID, directoryStoringProcessedFiles)

		return
	}

	//объединение списков файлов для задачи (возобновляемой или выполняемой на основе индексов)
	layoutListCompleted, err := common.MergingFileListForTaskFiltration(sma, mtfcJSON, clientID)
	if err != nil {
		errMsg.Info.ErrorName = "invalid value received"
		errMsg.Info.ErrorDescription = "Получено неверное значение, невозможно объединить список файлов, найденных в результате поиска по индексам"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		return
	}

	//если компоновка списка не завершена
	if !layoutListCompleted {
		return
	}

	modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID, directoryStoringProcessedFiles)
}
