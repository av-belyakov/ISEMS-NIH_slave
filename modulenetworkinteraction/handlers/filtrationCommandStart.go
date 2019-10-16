package handlers

/*
* Модуль выполняется при получении команды фильтрации 'start'
*
* Версия 0.1, дата релиза 14.05.2019
* */

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
	clientID, rootDirStoringFiles string) {

	saveMessageApp := savemessageapp.New()
	taskID := mtfcJSON.Info.TaskID
	np := common.NotifyParameters{
		ClientID: clientID,
		TaskID:   taskID,
		ChanRes:  cwtResText,
	}

	fmt.Printf("\tПринята задача по фильтрации сет. трафика с ID %v\n", taskID)

	rejectMsgJSON, err := json.Marshal(configure.MsgTypeFiltration{
		MsgType: "filtration",
		Info: configure.DetailInfoMsgFiltration{
			TaskID:     taskID,
			TaskStatus: "refused",
		},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	if mtfcJSON.Info.NumberMessagesFrom[0] == 0 {
		fmt.Println("\tпроверяем параметры фильтрации")

		//проверяем параметры фильтрации (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		if msg, ok := common.CheckParametersFiltration(&mtfcJSON.Info.Options); !ok {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

			if err := np.SendMsgNotify("danger", "filtration control", msg, "start"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		as := sma.GetApplicationSetting()

		fmt.Println("\tпроверяем наличие директорий переданных с сообщением типа 'Ping'")

		//проверяем наличие директорий переданных с сообщением типа 'Ping'
		newStorageFolders, err := checkExistDirectory(as.StorageFolders)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			d := "Не было задано ни одной директории для фильтрации сетевого трафика или заданные директории не были найденны. Задача отклонена."
			if err := np.SendMsgNotify("danger", "filtration control", d, "start"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			cwtResText <- configure.MsgWsTransmission{
				ClientID: clientID,
				Data:     &rejectMsgJSON,
			}

			return
		}

		fmt.Println("\tизменяем список директорий для фильтрации на реально существующие")

		//изменяем список директорий для фильтрации на реально существующие
		sma.SetApplicationSetting(configure.ApplicationSettings{
			TypeAreaNetwork: as.TypeAreaNetwork,
			StorageFolders:  newStorageFolders,
		})

		fmt.Println("\tи если параметры верны создаем новую задачу")

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
		fmt.Println("\tвыполнение фильтрации без индексов")

		go modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID, rootDirStoringFiles)

		return
	}

	//объединение списков файлов для задачи (выполняемой на основе индексов)
	layoutListCompleted, err := common.MergingFileListForTaskFiltration(sma, mtfcJSON, clientID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		d := "Получено неверное значение, невозможно объединить список файлов, найденных в результате поиска по индексам. Задача отклонена."
		if err := np.SendMsgNotify("danger", "filtration control", d, "start"); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
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

	go modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID, rootDirStoringFiles)
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
