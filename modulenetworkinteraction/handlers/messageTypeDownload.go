package handlers

/*
* Модуль обработчик запросов по скачиванию файла
*
* Версия 0.2, дата релиза 01.10.2019
* */

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

	fmt.Println("START function 'HandlerMessageTypeDownload'...")

	mtfcJSON := configure.MsgTypeDownloadControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	fmt.Printf("function 'HandlerMessageTypeDownload', RESIVED MSG '%v' FROM MASTER\n", mtfcJSON)

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
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

	//обработка сообщения 'готовность к приему файла'
	case "ready to receive file":
		if err := readyDownloadFile(np, sma, clientID, taskID, cwtResText, cwtResBinary); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

	//обработка запроса на останов выгрузки файла
	case "stop receiving files":
		if err := stopDownloadFile(np, sma, clientID, taskID, cwtResText); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

	//выполняем удаление файла при его успешной передаче
	case "file successfully accepted":

		fmt.Println("func 'handlerMessageTypeDownloadFile', MESSAGE: 'file successfully accepted'")

		//проверяем наличие задачи в 'StoreMemoryApplication'
		ti, err := sma.GetInfoTaskDownload(clientID, taskID)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		if err := os.Remove(path.Join(ti.DirectiryPathStorage, ti.FileName)); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//удаляем задачу
		if err := sma.DelTaskDownload(clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		}
	}
}
