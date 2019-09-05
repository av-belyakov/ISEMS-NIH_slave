package handlers

/*
* Модуль обработчик запросов по скачиванию файла
*
* Версия 0.1, дата релиза 05.09.2019
* */

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeDownload обработчик сообщения типа 'give me the file'
func HandlerMessageTypeDownload(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID, DirectoryStoringProcessedFiles string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission,
	cwtResBinary chan<- configure.MsgWsTransmission) {

	fmt.Println("START function 'HandlerMessageTypeDownload'...")

	mtfcJSON := configure.MsgTypeDownloadControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	taskID := mtfcJSON.Info.TaskID

	np := common.NotifyParameters{
		TaskID:   taskID,
		ClientID: clientID,
		ChanRes:  cwtResText,
	}

	rejectMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
		MsgType: "download control",
		Info: configure.DetailInfoMsgDownload{
			TaskID:  taskID,
			Command: "file transfer not possible",
		},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	switch mtfcJSON.Info.Command {
	//запрос на выгрузку файла
	case "give me the file":
		//проверяем параметры запроса
		msgErr, ok := common.CheckParametersDownloadFile(mtfcJSON.Info)
		if !ok {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for download file (client ID: %v, task ID: %v)", clientID, taskID))

			if err := np.SendMsgNotify("warning", "download control", msgErr, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		//проверяем наличие директории для хранения файла

		//проверяем наличие файла и его хеш-сумму

		//получаем кол-во частей файла

		//создаем новую задачу по выгрузке файла в 'StoreMemoryApplication'

		//отправляем сообщение 'ready for the transfer', кол-во частей файла и их
		// размер при успешной проверке

	//готовность к приему файла
	case "ready to receive file":
		//проверяем наличие задачи в 'StoreMemoryApplication'

		//запускаем передачу файла (добавляем в начале каждого файла строку типа '<id типа>:<id задачи>:<хеш файла>')

	//запрос на останов выгрузки файла
	case "stop receiving files":
		//проверяем наличие задачи в 'StoreMemoryApplication'

		//отправляем в канал полученный в разделе 'ready to receive file' запрос на останов чтения файла
		// отправляем 'file transfer stopped' при успешном останове передачи файла
	}
}
