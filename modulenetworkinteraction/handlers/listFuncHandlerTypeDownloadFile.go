package handlers

import (
	"encoding/json"
	"fmt"
	"path"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/moduledownloadfile"
	"ISEMS-NIH_slave/savemessageapp"
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
		msgErr := "источник сообщает - невозможно начать выгрузку файла, задача по скачиванию файла для данного клиента уже выполняется"

		fmt.Printf("func 'HandlerMessageTypeDownload' ERROR: %v\n", fmt.Sprint(err))

		err = np.SendMsgNotify("danger", "download files", msgErr, "stop")

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

		err = np.SendMsgNotify("danger", "download files", msgErr, "stop")

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
		errMsgHuman := "источник сообщает - невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе"

		fmt.Println("Невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе.")

		err = np.SendMsgNotify("danger", "download files", errMsgHuman, "stop")

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
				ChunkSize: maxSizeChunkFile,
			},
		},
	})
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

	//	fmt.Printf("func 'HandlerMessageTypeDownload', отправляем сообщение типа 'ready for the transfer' (%v) client with ID '%v', кол-во частей файла и их размер\n", msgRes, clientID)

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
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission,
	cwtResBinary chan<- configure.MsgWsTransmission) (err error) {

	fmt.Println("func 'HandlerMessageTypeDownload', MSG: '___ ready to receive file ___' проверяем наличие задачи в 'StoreMemoryApplication'")

	fn := "readyDownloadFile"

	rejectMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
		MsgType: "download files",
		Info: configure.DetailInfoMsgDownload{
			TaskID:  taskID,
			Command: "file transfer not possible",
		},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	//проверяем наличие задачи в 'StoreMemoryApplication'
	ti, err := sma.GetInfoTaskDownload(clientID, taskID)
	if err != nil {
		//		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("file download task not found for given ID (client ID: %v, task ID: %v)", clientID, taskID))
		msgErr := "источник сообщает - невозможно начать выгрузку файла, не найдена задача по выгрузке файла для заданного ID"

		err = np.SendMsgNotify("danger", "download files", msgErr, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &rejectMsgJSON,
		}

		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	fmt.Println("func 'HandlerMessageTypeDownload', запускаем передачу файла")

	chanStopReadFile := make(chan struct{})
	chanResponse := make(chan moduledownloadfile.TypeChannelMsgRes)

	//добавляем канал для останова чтения и передачи файла
	err = sma.AddChanStopReadFileTaskDownload(clientID, taskID, chanStopReadFile)

	//запускаем передачу файла (добавляем в начале каждого кусочка строку '<id тип передачи>:<id задачи>:<хеш файла>')
	go moduledownloadfile.ReadingFile(chanResponse,
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
		for message := range chanResponse {
			if message.ErrMsg != nil {
				fmt.Printf("func 'HandlerMessageTypeDownload', |||| ERROR: %v\n", fmt.Sprint(message.ErrMsg))

				_ = np.SendMsgNotify("danger", "download files", "выгрузка файла остановлена, ошибка чтения", "stop")

				cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &rejectMsgJSON,
				}

				continue
			}

			fmt.Printf("____ func 'HandlerMessageTypeDownload', RESEIVED MESSAGE FROM CHANNELL 'chanResponseError' (%v)\n", message.CauseStoped)

			//если задача полностью выполненна
			if message.CauseStoped == "completed" {
				if err := sma.SetIsCompletedTaskDownload(clientID, taskID); err != nil {
					_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})
				}

				continue
			}

			resMsg := configure.MsgTypeDownloadControl{
				MsgType: "download files",
				Info: configure.DetailInfoMsgDownload{
					TaskID:  taskID,
					Command: "file transfer stopped successfully",
				},
			}

			//если задача была завершена принудительно
			resMsgJSON, err := json.Marshal(resMsg)
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

			fmt.Printf("func 'handlerMessageTypeDownloadFile', sent MSG 'file transfer stopped successfully' MSG '%v' ----> Master\n", resMsg)

			//удаляем задачу так как она была принудительно остановлена
			_ = sma.DelTaskDownload(clientID, taskID)

		}
	}()

	return err
}

func createRejectMsgJSON(taskID string) (*[]byte, error) {
	rejectMsgJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
		MsgType: "download files",
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
