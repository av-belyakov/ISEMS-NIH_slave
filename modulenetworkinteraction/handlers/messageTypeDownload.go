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
	"ISEMS-NIH_slave/moduledownloadfile"
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

		fmt.Println("func 'HandlerMessageTypeDownload', проверяем, выполняется ли уже задача по выгрузке файла для данного клиента")

		//проверяем, выполняется ли задача по выгрузке файла для данного клиента
		if _, err := sma.GetInfoTaskDownload(clientID, taskID); err == nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("the download task for this client is already in progress (client ID: %v, task ID: %v)", clientID, taskID))
			msgErr := "Невозможно начать выгрузку файла, задача по скачиванию файла для данного клиента уже выполняется."

			fmt.Printf("func 'HandlerMessageTypeDownload' ERROR: %v\n", fmt.Sprint(err))

			if err := np.SendMsgNotify("danger", "download control", msgErr, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		fmt.Println("func 'HandlerMessageTypeDownload', проверяем параметры запроса")

		//проверяем параметры запроса
		msgErr, ok := common.CheckParametersDownloadFile(mtfcJSON.Info)
		if !ok {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for download file (client ID: %v, task ID: %v)", clientID, taskID))

			fmt.Printf("func 'HandlerMessageTypeDownload' ERROR: %v\n", fmt.Sprint(err))

			if err := np.SendMsgNotify("danger", "download control", msgErr, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		fmt.Println("func 'HandlerMessageTypeDownload', проверяем наличие файла, его размер и хеш-сумму")

		//проверяем наличие файла, его размер и хеш-сумму
		fileSize, fileHex, err := common.GetFileParameters(path.Join(mtfcJSON.Info.PathDirStorage, mtfcJSON.Info.FileOptions.Name))
		if (err != nil) || (fileSize != mtfcJSON.Info.FileOptions.Size) || (fileHex != mtfcJSON.Info.FileOptions.Hex) {
			errMsgLog := fmt.Sprintf("file upload cannot be started, the requested file is not found, or the file size and checksum do not match those accepted in the request (client ID: %v, task ID: %v)", clientID, taskID)
			errMsgHuman := "Невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе."

			fmt.Println("Невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе.")

			_ = saveMessageApp.LogMessage("error", errMsgLog)

			if err := np.SendMsgNotify("danger", "download control", errMsgHuman, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		strHex := fmt.Sprintf("1:%v:%v", taskID, fileHex)
		chunkSize := (appc.MaxSizeTransferredChunkFile - len(strHex))

		fmt.Println("func 'HandlerMessageTypeDownload', подсчитываем и получаем кол-во частей файла")

		//получаем кол-во частей файла
		numChunk := common.CountNumberParts(fileSize, chunkSize)
		responseMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
			MsgType: "download files",
			Info: configure.DetailInfoMsgDownload{
				TaskID:  taskID,
				Command: "ready for the transfer",
				FileOptions: configure.DownloadFileOptions{
					Name:      mtfcJSON.Info.FileOptions.Name,
					Size:      fileSize,
					Hex:       fileHex,
					NumChunk:  numChunk,
					ChunkSize: appc.MaxSizeTransferredChunkFile,
				},
			},
		})
		if err != nil {

			fmt.Printf("func 'HandlerMessageTypeDownload', %v\n", fmt.Sprint(err))

			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		fmt.Printf("func 'HandlerMessageTypeDownload', размер части = %v, всего частей = %v\n", chunkSize, numChunk)

		fmt.Println("func 'HandlerMessageTypeDownload', создаем новую задачу по выгрузке файла в 'StoreMemoryApplication'")

		//создаем новую задачу по выгрузке файла в 'StoreMemoryApplication'
		sma.AddTaskDownload(clientID, taskID, &configure.DownloadTasks{
			FileName:             mtfcJSON.Info.FileOptions.Name,
			FileSize:             fileSize,
			FileHex:              fileHex,
			NumFileChunk:         numChunk,
			SizeFileChunk:        appc.MaxSizeTransferredChunkFile,
			StrHex:               strHex,
			DirectiryPathStorage: mtfcJSON.Info.PathDirStorage,
		})

		tid, err := sma.GetInfoTaskDownload(clientID, taskID)

		fmt.Printf("func 'HandlerMessageTypeDownload', ERROR from GetInfoTaskDownload '%v'\n", fmt.Sprint(err))
		fmt.Printf("func 'HandlerMessageTypeDownload', Download Task Information: %v\n", tid)

		fmt.Println("func 'HandlerMessageTypeDownload', отправляем сообщение типа 'ready for the transfer', кол-во частей файла и их размер")

		//отправляем сообщение типа 'ready for the transfer', кол-во частей файла и их размер
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &responseMsgJSON,
		}

	//готовность к приему файла
	case "ready to receive file":

		fmt.Println("func 'HandlerMessageTypeDownload', проверяем наличие задачи в 'StoreMemoryApplication'")

		//проверяем наличие задачи в 'StoreMemoryApplication'
		ti, err := sma.GetInfoTaskDownload(clientID, taskID)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("file download task not found for given ID (client ID: %v, task ID: %v)", clientID, taskID))
			msgErr := "Невозможно начать выгрузку файла, не найдена задача по выгрузке файла для заданного ID."

			if err := np.SendMsgNotify("danger", "download control", msgErr, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		fmt.Println("func 'HandlerMessageTypeDownload', запускаем передачу файла")

		chanStopReadFile := make(chan struct{})

		//добавляем канал для останова чтения и передачи файла
		if err := sma.AddChanStopReadFileTaskDownload(clientID, taskID, chanStopReadFile); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//запускаем передачу файла (добавляем в начале каждого кусочка строку '<id тип передачи>:<id задачи>:<хеш файла>')
		err = moduledownloadfile.ReadingFile(moduledownloadfile.ReadingFileParameters{
			TaskID:           taskID,
			ClientID:         clientID,
			FileName:         ti.FileName,
			MaxChunkSize:     ti.SizeFileChunk,
			NumReadCycle:     ti.NumFileChunk,
			StrHex:           ti.StrHex,
			PathDirName:      ti.DirectiryPathStorage,
			ChanCWTResBinary: cwtResBinary,
		}, saveMessageApp, chanStopReadFile)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			msgErr := "Невозможно начать выгрузку файла, ошибка при чтении файла."

			fmt.Printf("func 'HandlerMessageTypeDownload', ERROR: %v\n", fmt.Sprint(err))

			if err := np.SendMsgNotify("danger", "download control", msgErr, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

	//запрос на останов выгрузки файла
	case "stop receiving files":
		//проверяем наличие задачи в 'StoreMemoryApplication'
		ti, err := sma.GetInfoTaskDownload(clientID, taskID)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("it is impossible to stop file transfer (client ID: %v, task ID: %v)", clientID, taskID))
			msgErr := "Невозможно остановить выгрузку файла, не найдена задача по выгрузке файла для заданного ID."

			fmt.Printf("func 'handlerMessageTypeDownloadFile', %v\n", msgErr)

			if err := np.SendMsgNotify("warning", "download control", msgErr, "stop"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			rejectTaskMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
				MsgType: "download control",
				Info: configure.DetailInfoMsgDownload{
					TaskID:  taskID,
					Command: "impossible to stop file transfer",
				},
			})
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

				return
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectTaskMsgJSON,
			}

			return
		}

		fmt.Println("func 'handlerMessageTypeDownloadFile', отправляем в канал полученный в разделе 'ready to receive file' запрос на останов чтения файла")

		//отправляем в канал полученный в разделе 'ready to receive file' запрос на останов чтения файла
		ti.ChanStopReadFile <- struct{}{}

	//выполняем удаление файла при его успешной передаче
	case "file successfully accepted":
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
