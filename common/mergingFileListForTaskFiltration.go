package common

import (
	"errors"
	"fmt"

	"ISEMS-NIH_slave/configure"
)

//MergingFileListForTaskFiltration выполняет объединение присылаемых клиентом списков
//файлов необходимых для выполнения фильтрации (данной действие выполняется для индексных списков)
func MergingFileListForTaskFiltration(
	sma *configure.StoreMemoryApplication,
	mtf *configure.MsgTypeFiltrationControl,
	clientID string) (bool, error) {

	if !mtf.Info.IndexIsFound {
		return true, errors.New("task filtering not index")
	}

	taskID := mtf.Info.TaskID

	//добавляем часть списка в осовной файл
	_, err := sma.AddFileToListFilesFiltrationTask(clientID, taskID, mtf.Info.ListFilesReceivedIndex)
	if err != nil {

		fmt.Printf("Function MergingFileListForTaskFiltration, ERROR: %v\n", err)

		return false, err
	}

	if mtf.Info.NumberMessagesFrom[0] == mtf.Info.NumberMessagesFrom[1] {
		return true, nil
	}

	return false, nil
}
