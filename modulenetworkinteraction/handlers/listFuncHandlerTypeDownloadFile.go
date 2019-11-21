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

	taskID := mtfcJSON.Info.TaskID

	rejectTaskMsgJSON, err := createRejectMsgJSON(taskID)
	if err != nil {
		return err
	}

	//проверяем, выполняется ли задача по выгрузке файла для данного клиента
	if _, err = sma.GetInfoTaskDownload(clientID, taskID); err == nil {
		msgErr := "источник сообщает - невозможно начать выгрузку файла, задача по скачиванию файла для данного клиента уже выполняется"
		err = np.SendMsgNotify("danger", "download files", msgErr, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	//проверяем параметры запроса
	msgErr, ok := common.CheckParametersDownloadFile(mtfcJSON.Info)
	if !ok {

		err = np.SendMsgNotify("danger", "download files", msgErr, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	//проверяем наличие файла, его размер и хеш-сумму
	fileSize, fileHex, err := common.GetFileParameters(path.Join(mtfcJSON.Info.PathDirStorage, mtfcJSON.Info.FileOptions.Name))
	if (err != nil) || (fileSize != mtfcJSON.Info.FileOptions.Size) || (fileHex != mtfcJSON.Info.FileOptions.Hex) {
		errMsgHuman := "источник сообщает - невозможно начать выгрузку файла, требуемый файл не найден или его размер и контрольная сумма не совпадают с принятыми в запросе"
		err = np.SendMsgNotify("danger", "download files", errMsgHuman, "stop")

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     rejectTaskMsgJSON,
		}

		return err
	}

	strHex := fmt.Sprintf("1:%v:%v", taskID, fileHex)
	chunkSize := (maxSizeChunkFile - len(strHex))

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
		return err
	}

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
				_ = np.SendMsgNotify("danger", "download files", "выгрузка файла остановлена, ошибка чтения", "stop")

				cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &rejectMsgJSON,
				}

				continue
			}

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
