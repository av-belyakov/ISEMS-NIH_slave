package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
*
* Версия 0.3, дата релиза 27.05.2019
* */

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//ChanDone содержит информацию о завершенной задаче
type chanDone struct {
	ClientID, TaskID, DirectoryName, TypeProcessing string
}

//ProcessingFiltration выполняет фильтрацию сет. трафика
func ProcessingFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID, rootDirStoringFiles string) {

	fmt.Println("START function 'ProcessingFiltration'...")

	saveMessageApp := savemessageapp.New()

	np := common.NotifyParameters{
		ClientID: clientID,
		TaskID:   taskID,
		ChanRes:  cwtResText,
	}

	d := "Инициализирована задача по фильтрации сетевого трафика, идет поиск файлов удовлетворяющих параметрам фильтрации"

	//информируем клиента о начале создания списка файлов удовлетворяющих параметрам фильтрации
	if err := np.SendMsgNotify("info", "filtration control", d, "start"); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//строим список файлов удовлетворяющих параметрам фильтрации
	if err := getListFilesForFiltering(sma, clientID, taskID); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		en := "cannot create a list of files"
		ed := "Ошибка, невозможно создать список файлов удовлетворяющий параметрам фильтрации"

		if err := np.SendMsgErr(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	//получаем информацию о выполняемой задачи
	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

		en := "invalid value received"
		ed := "Невозможно начать выполнение фильтрации, принят некорректный идентификатор клиента или задачи"

		if err := np.SendMsgErr(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	//проверяем количество файлов которые не были найдены при поиске их по индексам
	if info.NumberErrorProcessedFiles > 0 {
		d := "Внимание, фильтрация выполняется по файлам полученным при поиске по индексам. Однако, на диске были найдены не все файлы, перечисленные в индексах"

		if err := np.SendMsgNotify("warning", "filtration control", d, "start"); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}

	//поверяем количество файлов по которым необходимо выполнить фильтрацию
	if info.NumberFilesMeetFilterParameters == 0 {
		en := "no files matching configured interval"
		ed := "Внимание, фильтрация остановлена так как не найдены файлы удовлетворяющие параметрам фильтрации"

		if err := np.SendMsgErr(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	//создаем директорию для хранения отфильтрованных файлов
	if err := createDirectoryForFiltering(sma, clientID, taskID, rootDirStoringFiles); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		en := "unable to create directory"
		ed := "Ошибка при создании директории для хранения отфильтрованных файлов"

		if err := np.SendMsgErr(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	//создаем файл README с описанием параметров фильтрации
	if err := createFileReadme(sma, clientID, taskID); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		en := "cannot create a file README.txt"
		ed := "Невозможно создать файл с информацией о параметрах фильтрации"

		if err := np.SendMsgErr(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	as := sma.GetApplicationSetting()

	//формируем шаблон фильтрации
	patternScript := createPatternScript(info, as.TypeAreaNetwork)

	//изменяем статус задачи
	_ = sma.SetInfoTaskFiltration(clientID, taskID, map[string]interface{}{
		"Status": "execute",
	})

	done := make(chan chanDone, len(info.ListFiles))

	for dirName := range info.ListFiles {
		if len(info.ListFiles[dirName]) == 0 {
			continue
		}

		//запуск фильтрации для каждой директории
		go executeFiltration(done, info, sma, np, dirName, patternScript)
	}

	_ = saveMessageApp.LogMessage("info", fmt.Sprintf("start of a task to filter with the ID %v", taskID))

	//обработка информации о завершении фильтрации для каждой директории
	go filteringComplete(sma, np, done)
}

//выполнение фильтрации
func executeFiltration(
	done chan<- chanDone,
	ft *configure.FiltrationTasks,
	sma *configure.StoreMemoryApplication,
	np common.NotifyParameters,
	dirName, patternScript string) {

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

DONE:
	for _, file := range ft.ListFiles[dirName] {
		select {
		//выполнится если в канал придет запрос на останов фильтрации
		case <-ft.ChanStopFiltration:
			if err := sma.SetInfoTaskFiltration(np.ClientID, np.TaskID, map[string]interface{}{
				"Status": "stop",
			}); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			break DONE

		default:
			pathAndFileName := path.Join(dirName, file)
			patternScript = strings.Replace(patternScript, "$path_file_name", pathAndFileName, -1)

			//запускаем сформированный скрипт для поиска файлов
			if err := exec.Command("sh", "-c", patternScript).Run(); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprintf("%v\t%v, file: %v\n", err, dirName, file))

				//если ошибка увеличиваем количество обработанных с ошибкой файлов
				if _, err := sma.IncrementNumNotFoundIndexFiles(np.ClientID, np.TaskID); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
			}

			//увеличиваем кол-во обработанных файлов
			if _, err := sma.IncrementNumProcessedFiles(np.ClientID, np.TaskID); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprintf("%v\t%v, file: %v\n", err, dirName, file))
			}

			//если файл имеет размер больше 24 байта прибавляем его к найденным и складываем общий размер найденных файлов
			fileSize, fileHex, err := getFileParameters(pathAndFileName)

			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}

			if fileSize > int64(24) {
				if _, err := sma.IncrementNumFoundFiles(np.ClientID, np.TaskID, fileSize); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
			}

			taskInfo, err := sma.GetInfoTaskFiltration(np.ClientID, np.TaskID)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

				break DONE
			}

			msgRes := configure.MsgTypeFiltration{
				MsgType: "filtration",
				Info: configure.DetailInfoMsgFiltration{
					TaskID:                          np.TaskID,
					TaskStatus:                      taskInfo.Status,
					NumberFilesMeetFilterParameters: taskInfo.NumberFilesMeetFilterParameters,
					NumberProcessedFiles:            taskInfo.NumberProcessedFiles,
					NumberFilesFoundResultFiltering: taskInfo.NumberFilesFoundResultFiltering,
					NumberDirectoryFiltartion:       len(taskInfo.ListFiles),
					NumberErrorProcessedFiles:       taskInfo.NumberErrorProcessedFiles,
					SizeFilesMeetFilterParameters:   taskInfo.SizeFilesMeetFilterParameters,
					SizeFilesFoundResultFiltering:   taskInfo.SizeFilesFoundResultFiltering,
					PathStorageSource:               taskInfo.FileStorageDirectory,
					FoundFilesInformation: map[string]*configure.InputFilesInformation{
						file: &configure.InputFilesInformation{
							Size: fileSize,
							Hex:  fileHex,
						},
					},
				},
			}

			resJSON, err := json.Marshal(msgRes)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

				break DONE
			}

			//сообщение о ходе процесса фильтрации
			np.ChanRes <- configure.MsgWsTransmission{
				ClientID: np.ClientID,
				Data:     &resJSON,
			}
		}
	}

	sendInChan := chanDone{
		ClientID:       np.ClientID,
		TaskID:         np.TaskID,
		DirectoryName:  dirName,
		TypeProcessing: "complite",
	}

	ti, err := sma.GetInfoTaskFiltration(np.ClientID, np.TaskID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		done <- sendInChan

		return
	}

	sendInChan.TypeProcessing = ti.Status
	done <- sendInChan
}

//завершение выполнения фильтрации
func filteringComplete(sma *configure.StoreMemoryApplication, np common.NotifyParameters, done chan chanDone) {
	saveMessageApp := savemessageapp.New()

	defer close(done)

	var dirComplete int
	var responseDone chanDone

	taskInfo, err := sma.GetInfoTaskFiltration(np.ClientID, np.TaskID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	num := len(taskInfo.ListFiles)

	for dirComplete < num {
		responseDone = <-done
		if np.TaskID == responseDone.TaskID {
			dirComplete++
		}
	}

	if err := sma.SetInfoTaskFiltration(np.ClientID, np.TaskID, map[string]interface{}{
		"Status": responseDone.TypeProcessing,
	}); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//отправляем сообщение о завершении фильтрации и передаем СПИСОК ВСЕХ найденных в результате фильтрации файлов
	if err := SendMessageFiltrationComplete(np.ChanRes, sma, np.ClientID, np.TaskID); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	d := "задача по фильтрации сетевого трафика завершена"

	//информируем клиента об окончании фильтрации
	if err := np.SendMsgNotify("info", "filtration control", d, "complete"); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}
}
