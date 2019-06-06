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

	if mtfcJSON.Info.NumberMessagesFrom[0] == 0 {
		cs, ok := sma.GetClientSetting(clientID)
		if !ok {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("client with ID %v not found", clientID))

			en := "user error"
			ed := fmt.Sprintf("client with ID %v not found", clientID)
			if err := np.SendMsgErr(en, ed); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			return
		}

		mcpf := cs.MaxCountProcessFiltration

		tasksList, _ := sma.GetListTasksFiltration(clientID)

		//проверяем количество выполняемых задач (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		if len(tasksList) >= int(mcpf) {
			_ = saveMessageApp.LogMessage("info", "the maximum number of concurrent file filtering tasks has been reached, the task is rejected.")

			d := "Достигнут лимит максимального количества выполняемых, параллельных задач по фильтрации файлов. Задача отклонена."
			if err := np.SendMsgNotify("danger", "filtration control", d, "start"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			return
		}

		//проверяем параметры фильтрации (ТОЛЬКО ДЛЯ ПЕРВОГО СООБЩЕНИЯ)
		if msg, ok := common.CheckParametersFiltration(&mtfcJSON.Info.Options); !ok {
			_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

			if err := np.SendMsgNotify("danger", "filtration control", msg, "start"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			return
		}

		as := sma.GetApplicationSetting()

		//проверяем наличие директорий переданных с сообщением типа 'Ping'
		newStorageFolders, err := checkExistDirectory(as.StorageFolders)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			d := "Не было задано ни одной директории для фильтрации сетевого трафика или заданные директории не были найденны. Задача отклонена."
			if err := np.SendMsgNotify("danger", "filtration control", d, "start"); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
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
			ChanStopFiltration:              make(chan struct{}),
			ListFiles:                       make(map[string][]string),
		})
	}

	if !mtfcJSON.Info.IndexIsFound {
		go modulefiltrationfile.ProcessingFiltration(cwtResText, sma, clientID, taskID, rootDirStoringFiles)

		return
	}

	//объединение списков файлов для задачи (возобновляемой или выполняемой на основе индексов)
	layoutListCompleted, err := common.MergingFileListForTaskFiltration(sma, mtfcJSON, clientID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		d := "Получено неверное значение, невозможно объединить список файлов, найденных в результате поиска по индексам. Задача отклонена."
		if err := np.SendMsgNotify("danger", "filtration control", d, "start"); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//отправляем ответ на снятие задачи
		resJSON, err := json.Marshal(configure.MsgTypeFiltration{
			MsgType: "filtration",
			Info: configure.DetailInfoMsgFiltration{
				TaskID:     taskID,
				TaskStatus: "rejected",
			},
		})
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		//сообщение о ходе процесса фильтрации
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resJSON,
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
