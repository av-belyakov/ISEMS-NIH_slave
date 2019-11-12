package handlers

import (
	"encoding/json"
	"fmt"
	"path"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/moduledownloadfile"
)

func startDownloadFile(
	np common.NotifyParameters,
	sma *configure.StoreMemoryApplication,
	mtfcJSON configure.MsgTypeDownloadControl,
	clientID string,
	maxSizeChunkFile int,
	cwtResText chan<- configure.MsgWsTransmission) (err error) {

	fmt.Println("func 'HandlerMessageTypeDownload', проверяем, выполняется ли уже задача по выгрузке файла для данного клиента")

	taskID := mtfcJSON.Info.TaskID

	rejectTaskMsgJSON, err := createRejectMsgJSON(taskID)
	if err != nil {
		return err
	}

	//проверяем, выполняется ли задача по выгрузке файла для данного клиента
	if _, err = sma.GetInfoTaskDownload(clientID, taskID); err == nil {
		//		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("the download task for this client is already in progress (client ID: %v, task ID: %v)", clientID, taskID))
		msgErr := "Невозможно начать выгрузку файла, задача по скачиванию файла для данного клиента уже выполняется."

		fmt.Printf("func 'HandlerMessageTypeDownload' ERROR: %v\n", fmt.Sprint(err))

		err = np.SendMsgNotify("danger", "download control", msgErr, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	fmt.Println("func 'HandlerMessageTypeDownload', проверяем параметры запроса")

	//проверяем параметры запроса
	msgErr, ok := common.CheckParametersDownloadFile(mtfcJSON.Info)
	if !ok {
		//_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for download file (client ID: %v, task ID: %v)", clientID, taskID))

		fmt.Printf("func 'HandlerMessageTypeDownload' ERROR: %v\n", fmt.Sprint(err))

		err = np.SendMsgNotify("danger", "download control", msgErr, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	fmt.Println("func 'HandlerMessageTypeDownload', проверяем наличие файла, его размер и хеш-сумму")

	//проверяем наличие файла, его размер и хеш-сумму
	fileSize, fileHex, err := common.GetFileParameters(path.Join(mtfcJSON.Info.PathDirStorage, mtfcJSON.Info.FileOptions.Name))
	if (err != nil) || (fileSize != mtfcJSON.Info.FileOptions.Size) || (fileHex != mtfcJSON.Info.FileOptions.Hex) {
		//errMsgLog := fmt.Sprintf("file upload cannot be started, the requested file is not found, or the file size and checksum do not match those accepted in the request (client ID: %v, task ID: %v)", clientID, taskID)
		errMsgHuman := "Невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе."

		fmt.Println("Невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе.")

		err = np.SendMsgNotify("danger", "download control", errMsgHuman, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	strHex := fmt.Sprintf("1:%v:%v", taskID, fileHex)
	chunkSize := (maxSizeChunkFile - len(strHex))

	fmt.Println("func 'HandlerMessageTypeDownload', подсчитываем и получаем кол-во частей файла")

	//получаем кол-во частей файла
	numChunk := common.CountNumberParts(fileSize, chunkSize)
	msgRes := configure.MsgTypeDownloadControl{
		MsgType: "download files",
		Info: configure.DetailInfoMsgDownload{
			TaskID:  taskID,
			Command: "ready for the transfer",
			FileOptions: configure.DownloadFileOptions{
				Name:      mtfcJSON.Info.FileOptions.Name,
				Size:      fileSize,
				Hex:       fileHex,
				NumChunk:  numChunk,
				ChunkSize: maxSizeChunkFile,
			},
		},
	}

	responseMsgJSON, err := json.Marshal(msgRes)
	if err != nil {

		fmt.Printf("func 'HandlerMessageTypeDownload', %v\n", fmt.Sprint(err))

		return err
	}

	fmt.Printf("func 'HandlerMessageTypeDownload', размер части = %v, всего частей = %v\n", chunkSize, numChunk)

	fmt.Println("func 'HandlerMessageTypeDownload', создаем новую задачу по выгрузке файла в 'StoreMemoryApplication'")

	//создаем новую задачу по выгрузке файла в 'StoreMemoryApplication'
	sma.AddTaskDownload(clientID, taskID, &configure.DownloadTasks{
		FileName:             mtfcJSON.Info.FileOptions.Name,
		FileSize:             fileSize,
		FileHex:              fileHex,
		NumFileChunk:         numChunk,
		SizeFileChunk:        maxSizeChunkFile,
		StrHex:               strHex,
		DirectiryPathStorage: mtfcJSON.Info.PathDirStorage,
	})

	tid, err := sma.GetInfoTaskDownload(clientID, taskID)

	fmt.Printf("func 'HandlerMessageTypeDownload', ERROR from GetInfoTaskDownload '%v'\n", fmt.Sprint(err))
	fmt.Printf("func 'HandlerMessageTypeDownload', Download Task Information: %v\n", tid)

	fmt.Printf("func 'HandlerMessageTypeDownload', отправляем сообщение типа 'ready for the transfer' (%v) client with ID '%v', кол-во частей файла и их размер\n", msgRes, clientID)

	//отправляем сообщение типа 'ready for the transfer', кол-во частей файла и их размер
	cwtResText <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &responseMsgJSON,
	}

	return err
}

func readyDownloadFile(
	np common.NotifyParameters,
	sma *configure.StoreMemoryApplication,
	clientID string,
	taskID string,
	cwtResText chan<- configure.MsgWsTransmission,
	cwtResBinary chan<- configure.MsgWsTransmission) (err error) {

	fmt.Println("func 'HandlerMessageTypeDownload', MSG: '___ ready to receive file ___' проверяем наличие задачи в 'StoreMemoryApplication'")

	rejectMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
		MsgType: "download control",
		Info: configure.DetailInfoMsgDownload{
			TaskID:  taskID,
			Command: "file transfer not possible",
		},
	})
	if err != nil {
		return err
	}

	//проверяем наличие задачи в 'StoreMemoryApplication'
	ti, err := sma.GetInfoTaskDownload(clientID, taskID)
	if err != nil {
		//		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("file download task not found for given ID (client ID: %v, task ID: %v)", clientID, taskID))
		msgErr := "Невозможно начать выгрузку файла, не найдена задача по выгрузке файла для заданного ID."

		err = np.SendMsgNotify("danger", "download control", msgErr, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &rejectMsgJSON,
		}

		return err
	}

	fmt.Println("func 'HandlerMessageTypeDownload', запускаем передачу файла")

	chanStopReadFile := make(chan struct{})
	chanResponseError := make(chan error)

	//добавляем канал для останова чтения и передачи файла
	err = sma.AddChanStopReadFileTaskDownload(clientID, taskID, chanStopReadFile)

	//запускаем передачу файла (добавляем в начале каждого кусочка строку '<id тип передачи>:<id задачи>:<хеш файла>')
	go moduledownloadfile.ReadingFile(chanResponseError,
		moduledownloadfile.ReadingFileParameters{
			TaskID:           taskID,
			ClientID:         clientID,
			StrHex:           ti.StrHex,
			FileName:         ti.FileName,
			NumReadCycle:     ti.NumFileChunk,
			MaxChunkSize:     ti.SizeFileChunk,
			PathDirName:      ti.DirectiryPathStorage,
			ChanCWTResBinary: cwtResBinary,
		}, chanStopReadFile)

	go func() {
		msgErr := "Выгрузка файла остановлена, ошибка чтения."

		for errMessage := range chanResponseError {
			if errMessage != nil {
				//				_ = saveMessageApp.LogMessage("error", fmt.Sprint(errMessage))

				fmt.Printf("func 'HandlerMessageTypeDownload', |||| ERROR: %v\n", fmt.Sprint(errMessage))

				/*if err := np.SendMsgNotify("danger", "download control", msgErr, "stop"); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}*/

				_ = np.SendMsgNotify("danger", "download control", msgErr, "stop")

				cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &rejectMsgJSON,
				}

				continue
			}

			fmt.Printf("func 'HandlerMessageTypeDownload', reseived message from channel 'chanResponseError' (%v)\n", errMessage)

			//фактически удаляем канал для останова чтения и передачи файла
			_ = sma.CloseChanStopReadFileTaskDownload(clientID, taskID)
		}
	}()

	return err
}

func stopDownloadFile(
	np common.NotifyParameters,
	sma *configure.StoreMemoryApplication,
	clientID string,
	taskID string,
	cwtResText chan<- configure.MsgWsTransmission) (err error) {

	fmt.Println("\t_______func 'handlerMessageTypeDownloadFile', запрос на останов выгрузки файла")

	//проверяем наличие задачи в 'StoreMemoryApplication'
	ti, err := sma.GetInfoTaskDownload(clientID, taskID)
	if err != nil {
		//_ = saveMessageApp.LogMessage("error", fmt.Sprintf("it is impossible to stop file transfer (client ID: %v, task ID: %v)", clientID, taskID))
		msgErr := "Невозможно остановить выгрузку файла, не найдена задача по выгрузке файла для заданного ID."

		fmt.Printf("func 'handlerMessageTypeDownloadFile', ERROR: %v\n", msgErr)

		err = np.SendMsgNotify("warning", "download control", msgErr, "stop")

		rejectTaskMsgJSON, err := createRejectMsgJSON(taskID)
		if err != nil {
			return err
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	fmt.Println("func 'handlerMessageTypeDownloadFile', отправляем в канал полученный в разделе 'ready to receive file' запрос на останов чтения файла 111")
	fmt.Printf("_-_-_-_-_-_- func 'handlerMessageTypeDownloadFile', CHAN STOP - '%v'\n", ti.ChanStopReadFile)

	if ti.ChanStopReadFile != nil {
		//отправляем в канал полученный в разделе 'ready to receive file' запрос на останов чтения файла
		ti.ChanStopReadFile <- struct{}{}

		fmt.Println("func 'handlerMessageTypeDownloadFile', отправляем в канал полученный в разделе 'ready to receive file' запрос на останов чтения файла 222")
	}

	fmt.Printf("func 'handlerMessageTypeDownloadFile', DELETE TASK WITH ID:'%v'\n", taskID)

	//удаляем задачу
	err = sma.DelTaskDownload(clientID, taskID)

	tti, _ := sma.GetInfoTaskDownload(clientID, taskID)
	fmt.Printf("|||||||||||------- func 'handlerMessageTypeDownloadFile', DELETE task '%v' in succesfull\n", tti)

	return err
}

func createRejectMsgJSON(taskID string) (*[]byte, error) {
	rejectMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
		MsgType: "download control",
		Info: configure.DetailInfoMsgDownload{
			TaskID:  taskID,
			Command: "file transfer not possible",
		},
	})
	if err != nil {
		return nil, err
	}

	return &rejectMsgJSON, nil
}
