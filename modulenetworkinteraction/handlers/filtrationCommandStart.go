package handlers

/*
* Модуль выполняется при получении команды фильтрации 'start'
*
* Версия 0.1, дата релиза 14.05.2019
* */

import (
	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulefiltrationfile"
	"ISEMS-NIH_slave/savemessageapp"
	"encoding/json"
	"fmt"
)

//StartFiltration обработчик команды 'start' раздела фильтрации
func StartFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	mtfcJSON *configure.MsgTypeFiltrationControl,
	clientID, taskID string) {

	saveMessageApp := savemessageapp.New()

	if mtfcJSON.Info.NumberMessagesFrom[0] == 0 {
		//проверяем количество выполняемых задач (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		cs, ok := sma.GetClientSetting(clientID)
		if !ok {
			_ = saveMessageApp.LogMessage("error", "client with ID "+clientID+" not found")

			return
		}

		mcpf := cs.MaxCountProcessFiltration

		tasksList, _ := sma.GetListTasksFiltration(clientID)

		//проверяем количество выполняемых задач по фильтрации
		if len(tasksList) >= int(mcpf) {
			msgJSON, err := json.Marshal(configure.MsgTypeError{
				MsgType: "error",
				Info: configure.DetailInfoMsgError{
					TaskID:           mtfcJSON.Info.TaskID,
					ErrorName:        "max limit task filtration",
					ErrorDescription: "the maximum limit of concurrent tasks filter",
				},
			})
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
			msgJSON, err := json.Marshal(configure.MsgTypeError{
				MsgType: "error",
				Info: configure.DetailInfoMsgError{
					TaskID:           mtfcJSON.Info.TaskID,
					ErrorName:        "invalid value received",
					ErrorDescription: msg,
				},
			})
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

				return
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &msgJSON,
			}

			_ = saveMessageApp.LogMessage("error", "incorrect parameters for filtering (client ID: "+clientID+", task ID: "+mtfcJSON.Info.TaskID+")")

			return
		}

		//и если параметры верны создаем новую задачу
		sma.AddTaskFiltration(clientID, taskID, &configure.FiltrationTasks{
			DateTimeStart:   mtfcJSON.Info.Options.DateTime.Start,
			DateTimeEnd:     mtfcJSON.Info.Options.DateTime.End,
			Protocol:        mtfcJSON.Info.Options.Protocol,
			UseIndex:        mtfcJSON.Info.IndexIsFound,
			CountIndexFiles: mtfcJSON.Info.CountIndexFiles,
			Filters:         mtfcJSON.Info.Options.Filters,
			ListFiles:       make(map[string][]string),
		})

		/*
			!!! ВАЖНО !!!
				Необходимо:
				1. доделать проверку паредавемых пользователем параметров,
				2. при успешной проверке сделать добавление информации о задаче в StoringMemoryTask,
				3. доделать объединение списка файлов найденных в результате поиска по индексам
				4. выполнить тестирование всех выше перечисленых пунктов
		*/
	}

	if mtfcJSON.Info.IndexIsFound {
		//объединение списков файлов для задачи (возобновляемой или выполняемой на основе индексов)
		layoutListCompleted, err := common.MergingFileListForTaskFiltration(sma, mtfcJSON, clientID, taskID)
		if err != nil {
			msgJSON, err := json.Marshal(configure.MsgTypeError{
				MsgType: "error",
				Info: configure.DetailInfoMsgError{
					TaskID:           mtfcJSON.Info.TaskID,
					ErrorName:        "invalid value received",
					ErrorDescription: "an incorrect value is received, it is impossible to merge the list of files found by indexes",
				},
			})
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

		//выполняем фильтрацию (для списка файов полученных в результате поиска по индексам)
		modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID)
	}

	//выполняем фильтрацию
	modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID)
}
