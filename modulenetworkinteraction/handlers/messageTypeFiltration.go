package handlers

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeFiltration обработчик сообщений типа 'filtration'
func HandlerMessageTypeFiltration(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID, directoryStoringProcessedFiles string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission) {

	fn := "HandlerMessageTypeFiltration"

	mtfcJSON := configure.MsgTypeFiltrationControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	np := common.NotifyParameters{
		TaskID:   mtfcJSON.Info.TaskID,
		ClientID: clientID,
		ChanRes:  cwtResText,
	}

	if mtfcJSON.Info.Command == "start" {
		go StartFiltration(cwtResText, sma, &mtfcJSON, clientID, directoryStoringProcessedFiles, saveMessageApp)
	}

	if mtfcJSON.Info.Command == "stop" {
		task, err := sma.GetInfoTaskFiltration(clientID, mtfcJSON.Info.TaskID)
		if err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

			d := "источник сообщает - невозможно остановить выполнение фильтрации, не найдена задача с заданным идентификатором"
			if err := np.SendMsgNotify("warning", "filtration control", d, "stop"); err != nil {
				saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}

			return
		}

		if err := sma.SetInfoTaskFiltration(np.ClientID, np.TaskID, map[string]interface{}{"Status": "stop"}); err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем запрос на останов задачи по фильтрации файлов
		//task.ChanStopFiltration <- struct{}{}
		num := 1
		for _, c := range task.ListChanStopFiltration {
			c <- struct{}{}

			num++
		}
	}

	//при получении подтверждения о завершении фильтрации удаляем задачу
	if mtfcJSON.Info.Command == "confirm complete" {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			TypeMessage: "info",
			Description: fmt.Sprintf("deleting a filter task with ID '%v' because the task was completed", mtfcJSON.Info.TaskID),
			FuncName:    fn,
		})

		if err := sma.DelTaskFiltration(clientID, mtfcJSON.Info.TaskID); err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}
	}
}
