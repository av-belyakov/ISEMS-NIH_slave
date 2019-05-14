package common

import (
	"errors"

	"ISEMS-NIH_slave/configure"
)

//MergingFileListForTaskFiltration выполняет объединение присылаемых клиентом списков
//файлов необходимых для выполнения фильтрации (данной действие выполняется для индексных списков)
func MergingFileListForTaskFiltration(
	sma *configure.StoreMemoryApplication,
	mtf *configure.MsgTypeFiltrationControl,
	clientID, taskID string) (bool, error) {

	if !mtf.Info.IndexIsFound {
		return true, errors.New("task filtering not index")
	}

	/*if mtf.Info.Settings.CountPartsIndexFiles[0] == 0 {
		ift.TaskID[mtf.Info.TaskIndex].TotalNumberFilesFilter = mtf.Info.Settings.TotalNumberFilesFilter
		ift.TaskID[mtf.Info.TaskIndex].UseIndexes = true

		ift.TaskID[mtf.Info.TaskIndex].NumberPleasantMessages++

		return false, nil
	}

	var countFiles, fullCountFiles int
	for dir, files := range mtf.Info.Settings.ListFilesFilter {
		ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir] = append(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir], files...)

		countFiles += len(files)
		fullCountFiles += len(ift.TaskID[mtf.Info.TaskIndex].ListFilesFilter[dir])
	}

	if mtf.Info.Settings.CountPartsIndexFiles[0] == mtf.Info.Settings.CountPartsIndexFiles[1] {
		return true, nil
	}
	*/

	//добавляем часть списка в осовной файл
	if _, err := sma.AddFileToListFilesFiltrationTask(clientID, taskID, mtf.Info.ListFilesReceivedIndex); err != nil {
		return false, err
	}

	//получаем параметры задачи
	taskInfo, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return false, err
	}

	var fullCountFiles int
	//считаем общее количество файлов в списке
	for _, v := range taskInfo.ListFiles {
		fullCountFiles += len(v)
	}

	if (mtf.Info.NumberMessagesFrom[0] == mtf.Info.NumberMessagesFrom[1]) && (taskInfo.CountIndexFiles == fullCountFiles) {
		return true, nil
	}

	return false, nil
}
