package handlers

import (
	"encoding/json"
	"fmt"
	"moth_go/helpers"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeFiltration обработчик сообщений типа 'Filtration'
func HandlerMessageTypeFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string) {

	fmt.Println("START function 'HandlerMessageTypeFiltration'...")

	saveMessageApp := savemessageapp.New()

	mtfcJSON := configure.MsgTypeFiltrationControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	if mtfcJSON.Info.NumberMessagesFrom[1] == 0 {
		//проверяем количество выполняемых задач (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		cs, ok := sma.GetClientSetting(clientID)
		if !ok {
			_ = saveMessageApp.LogMessage("error", "client with ID "+clientID+" not found")

			return
		}

		mcpf := cs.MaxCountProcessFiltration

		tasksList, _ := sma.GetListTasksFiltration(clientID)
		if len(tasksList) >= int(mcpf) {
			//если количество задач превышает установленное значение,
			//то отправляем сообщение об ошибке
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
		//и если параметры верны создаем новую задачу
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
		layoutListCompleted, err := helpers.MergingFileListForTaskFilter(sma, &mtfcJSON)
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
	}
}
