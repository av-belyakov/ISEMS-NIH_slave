package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulefiltrationfile"
	"ISEMS-NIH_slave/savemessageapp"
)

//StartFiltration обработчик команды 'start' раздела фильтрации
func StartFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	mtfcJSON *configure.MsgTypeFiltrationControl,
	clientID, rootDirStoringFiles string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	taskID := mtfcJSON.Info.TaskID
	np := common.NotifyParameters{
		ClientID: clientID,
		TaskID:   taskID,
		ChanRes:  cwtResText,
	}
	fn := "StartFiltration"

	rejectMsgJSON, err := json.Marshal(configure.MsgTypeFiltration{
		MsgType: "filtration",
		Info: configure.DetailInfoMsgFiltration{
			TaskID:     taskID,
			TaskStatus: "refused",
		},
	})
	if err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	if mtfcJSON.Info.NumberMessagesFrom[0] == 0 {
		//проверяем параметры фильтрации (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		if msg, ok := common.CheckParametersFiltration(&mtfcJSON.Info.Options); !ok {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID),
				FuncName:    fn,
			})

			if err := np.SendMsgNotify("danger", "filtration control", msg, "stop"); err != nil {
				saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		as := sma.GetApplicationSetting()

		//проверяем наличие директорий переданных с сообщением типа 'Ping'
		newStorageFolders, err := checkExistDirectory(as.StorageFolders)
		if err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

			d := "источник сообщает - не было задано ни одной директории для фильтрации сетевого трафика или заданные директории не были найденны"
			if err := np.SendMsgNotify("danger", "filtration control", d, "stop"); err != nil {
				saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		//изменяем список директорий для фильтрации на реально существующие
		sma.SetApplicationSetting(configure.ApplicationSettings{
			TypeAreaNetwork: as.TypeAreaNetwork,
			StorageFolders:  newStorageFolders,
		})

		//и если параметры верны создаем новую задачу
		sma.AddTaskFiltration(clientID, taskID, &configure.FiltrationTasks{
			DateTimeStart:                   mtfcJSON.Info.Options.DateTime.Start,
			DateTimeEnd:                     mtfcJSON.Info.Options.DateTime.End,
			Protocol:                        mtfcJSON.Info.Options.Protocol,
			UseIndex:                        mtfcJSON.Info.IndexIsFound,
			NumberFilesMeetFilterParameters: mtfcJSON.Info.CountIndexFiles,
			Filters:                         mtfcJSON.Info.Options.Filters,
			ListChanStopFiltration:          make([]chan struct{}, 0, len(as.StorageFolders)),
			ListFiles:                       make(map[string][]string),
		})
	}

	if !mtfcJSON.Info.IndexIsFound {
		go modulefiltrationfile.ProcessingFiltration(cwtResText, sma, saveMessageApp, clientID, taskID, rootDirStoringFiles)

		return
	}

	//объединение списков файлов для задачи (выполняемой на основе индексов)
	layoutListCompleted, err := common.MergingFileListForTaskFiltration(sma, mtfcJSON, clientID)
	if err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		d := "источник сообщает - получено неверное значение, невозможно объединить список файлов, найденных в результате поиска по индексам"
		if err := np.SendMsgNotify("danger", "filtration control", d, "stop"); err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &rejectMsgJSON,
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	//если компоновка списка не завершена
	if !layoutListCompleted {
		return
	}

	go modulefiltrationfile.ProcessingFiltration(cwtResText, sma, saveMessageApp, clientID, taskID, rootDirStoringFiles)
}

func checkExistDirectory(lf []string) ([]string, error) {
	newList := make([]string, 0, len(lf))

	if len(lf) == 0 {
		return newList, errors.New("was not asked a single directory for network traffic filtering")
	}

	numDIr := len(lf)
	for _, d := range lf {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			numDIr--

			continue
		}

		newList = append(newList, d)
	}

	if numDIr == 0 {
		return newList, errors.New("invalid directory names are defined for network traffic filtering")
	}

	return newList, nil
}
