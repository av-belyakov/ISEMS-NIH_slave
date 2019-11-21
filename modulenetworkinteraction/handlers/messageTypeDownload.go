package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeDownload обработчик сообщения типа 'give me the file'
func HandlerMessageTypeDownload(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string,
	appc *configure.AppConfig,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission,
	cwtResBinary chan<- configure.MsgWsTransmission) {

	fn := "HandlerMessageTypeDownload"

	mtfcJSON := configure.MsgTypeDownloadControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	taskID := mtfcJSON.Info.TaskID

	np := common.NotifyParameters{
		TaskID:   taskID,
		ClientID: clientID,
		ChanRes:  cwtResText,
	}

	switch mtfcJSON.Info.Command {
	//обработка запроса на выгрузку файла
	case "give me the file":
		if err := startDownloadFile(np, sma, mtfcJSON, clientID, appc.MaxSizeTransferredChunkFile, cwtResText); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

	//обработка сообщения 'готовность к приему файла'
	case "ready to receive file":
		readyDownloadFile(np, sma, clientID, taskID, saveMessageApp, cwtResText, cwtResBinary)

	//обработка запроса на останов выгрузки файла
	case "stop receiving files":
		//проверяем наличие задачи в 'StoreMemoryApplication'
		ti, err := sma.GetInfoTaskDownload(clientID, taskID)
		if err != nil {
			return
		}

		//если задача не была завершена автоматически по мере выполнения
		if !ti.IsTaskCompleted {
			if ti.ChanStopReadFile != nil {
				ti.ChanStopReadFile <- struct{}{}

				break
			}
		}

		//если задача была выполненна полностью но MASTER считает что задача должна быть остановлена
		resMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
			MsgType: "download files",
			Info: configure.DetailInfoMsgDownload{
				TaskID:  taskID,
				Command: "file transfer stopped successfully",
			},
		})
		if err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resMsgJSON,
		}

		//удаляем задачу так как она была принудительно остановлена
		_ = sma.DelTaskDownload(clientID, taskID)

	//выполняем удаление файла при его успешной передаче
	case "file successfully accepted":
		//проверяем наличие задачи в 'StoreMemoryApplication'
		ti, err := sma.GetInfoTaskDownload(clientID, taskID)
		if err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

			break
		}

		if err := os.Remove(path.Join(ti.DirectiryPathStorage, ti.FileName)); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//удаляем задачу
		if err := sma.DelTaskDownload(clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//выполняем удаление задачи при неуспешной передачи файла
	case "file received with error":
		//удаляем задачу
		if err := sma.DelTaskDownload(clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}
	}
}
