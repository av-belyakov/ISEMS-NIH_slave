package handlers

/*
* Модуль обработчик запросов по фильтрации
*
* Версия 0.1, дата релиза 15.05.2019
* */

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeFiltration обработчик сообщений типа 'Filtration'
func HandlerMessageTypeFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID, directoryStoringProcessedFiles string) {

	fmt.Println("START function 'HandlerMessageTypeFiltration'...")

	saveMessageApp := savemessageapp.New()

	mtfcJSON := configure.MsgTypeFiltrationControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		msgJSON, err := json.Marshal(configure.MsgTypeError{
			MsgType: "error",
			Info: configure.DetailInfoMsgError{
				TaskID:           mtfcJSON.Info.TaskID,
				ErrorName:        "invalid value received",
				ErrorDescription: "Принят не валидный тип JSON сообщения",
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

	if mtfcJSON.Info.Command == "start" {
		go StartFiltration(cwtResText, sma, &mtfcJSON, clientID, directoryStoringProcessedFiles)
	}

	if mtfcJSON.Info.Command == "stop" {
		task, err := sma.GetInfoTaskFiltration(clientID, mtfcJSON.Info.TaskID)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			msgJSON, err := json.Marshal(configure.MsgTypeError{
				MsgType: "error",
				Info: configure.DetailInfoMsgError{
					TaskID:           mtfcJSON.Info.TaskID,
					ErrorName:        "invalid value received",
					ErrorDescription: "Невозможно остановить выполнение фильтрации, не найдена задача с заданным идентификатором",
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

		//отправляем запрос на останов задачи по фильтрации файлов
		task.ChanStopFiltration <- struct{}{}
	}

	//при получении подтверждения о завершении фильтрации (не важно 'stop' или 'complite') удаляем задачу
	if mtfcJSON.Info.Command == "confirm complite" {
		if err := sma.DelTaskFiltration(clientID, mtfcJSON.Info.TaskID); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}

}
