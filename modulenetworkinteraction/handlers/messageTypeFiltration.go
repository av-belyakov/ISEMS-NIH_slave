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
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
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
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

			d := "источник сообщает - невозможно остановить выполнение фильтрации, не найдена задача с заданным идентификатором"
			if err := np.SendMsgNotify("warning", "filtration control", d, "stop"); err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}

			return
		}

		fmt.Println("func 'messageTypeFiltration' RESIVED MSG 'STOP FILTRATION'")

		if err := sma.SetInfoTaskFiltration(np.ClientID, np.TaskID, map[string]interface{}{"Status": "stop"}); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем запрос на останов задачи по фильтрации файлов
		//task.ChanStopFiltration <- struct{}{}
		num := 1
		for _, c := range task.ListChanStopFiltration {
			fmt.Printf("func 'messageTypeFiltration' SEND msg to CHAN STOP %v\n", num)

			c <- struct{}{}

			num++
		}
	}

	//при получении подтверждения о завершении фильтрации удаляем задачу
	if mtfcJSON.Info.Command == "confirm complete" {
		fmt.Println("function 'messageTypeFiltration', resived message type 'confirm complete'")

		if err := sma.DelTaskFiltration(clientID, mtfcJSON.Info.TaskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}
	}
}
