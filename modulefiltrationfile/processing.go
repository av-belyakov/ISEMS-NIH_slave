package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
* */

import (
	"encoding/json"
	"fmt"
	"os"
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
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	clientID, taskID, rootDirStoringFiles string) {

	np := common.NotifyParameters{
		ClientID: clientID,
		TaskID:   taskID,
		ChanRes:  cwtResText,
	}
	fn := "ProcessingFiltration"

	d := "источник сообщает - задача инициализирована, идет поиск файлов удовлетворяющих параметрам фильтрации"
	if err := np.SendMsgNotify("info", "filtration control", d, "start"); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	fmt.Println("//строим список файлов удовлетворяющих параметрам фильтрации")

	//строим список файлов удовлетворяющих параметрам фильтрации
	if err := getListFilesForFiltering(sma, clientID, taskID); err != nil {

		fmt.Printf("func 'ProcessingFiltration' ERROR: %v\n", err)

		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		d := "источник сообщает - невозможно создать список файлов удовлетворяющий параметрам фильтрации"
		if err := np.SendMsgNotify("danger", "filtration control", d, "stop"); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем ответ на снятие задачи
		if err := sendMsgTypeFilteringRefused(cwtResText, clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	fmt.Println("//получаем информацию о выполняемой задаче")

	//получаем информацию о выполняемой задачи
	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID),
			FuncName:    fn,
		})

		d := "источник сообщает - невозможно начать выполнение фильтрации, принят некорректный идентификатор клиента или задачи"
		if err := np.SendMsgNotify("danger", "filtration control", d, "stop"); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем ответ на снятие задачи
		if err := sendMsgTypeFilteringRefused(cwtResText, clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	fmt.Println("//проверяем количество файлов которые не были найдены при поиске их по индексам")

	//проверяем количество файлов которые не были найдены при поиске их по индексам
	if info.NumberErrorProcessedFiles > 0 {
		d := "источник сообщает - внимание, фильтрация выполняется по файлам полученным при поиске по индексам. Однако, на диске были найдены не все файлы, перечисленные в индексах"
		if err := np.SendMsgNotify("warning", "filtration control", d, "start"); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}
	}

	fmt.Println("//поверяем количество файлов по которым необходимо выполнить фильтрацию")

	//поверяем количество файлов по которым необходимо выполнить фильтрацию
	if info.NumberFilesMeetFilterParameters == 0 {
		d := "источник сообщает - внимание, фильтрация остановлена так как не найдены файлы удовлетворяющие заданным параметрам"
		if err := np.SendMsgNotify("warning", "filtration control", d, "stop"); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем ответ на снятие задачи
		if err := sendMsgTypeFilteringRefused(cwtResText, clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	fmt.Println("//создаем директорию для хранения отфильтрованных файлов")

	//создаем директорию для хранения отфильтрованных файлов
	if err := createDirectoryForFiltering(sma, clientID, taskID, rootDirStoringFiles); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		d := "источник сообщает - ошибка при создании директории для хранения отфильтрованных файлов"
		if err := np.SendMsgNotify("danger", "filtration control", d, "stop"); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем ответ на снятие задачи
		if err := sendMsgTypeFilteringRefused(cwtResText, clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	fmt.Println("//создаем файл README с описанием параметров фильтрации")

	//создаем файл README с описанием параметров фильтрации
	if err := createFileReadme(sma, clientID, taskID); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		d := "источник сообщает - невозможно создать файл с информацией о параметрах фильтрации"
		if err := np.SendMsgNotify("danger", "filtration control", d, "stop"); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//отправляем ответ на снятие задачи
		if err := sendMsgTypeFilteringRefused(cwtResText, clientID, taskID); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//удаляем задачу
		sma.DelTaskFiltration(clientID, taskID)

		return
	}

	as := sma.GetApplicationSetting()

	fmt.Println("//формируем шаблон фильтрации")

	//формируем шаблон фильтрации
	patternScript := createPatternScript(info, as.TypeAreaNetwork)

	fmt.Println("//изменяем статус задачи")

	//изменяем статус задачи
	_ = sma.SetInfoTaskFiltration(clientID, taskID, map[string]interface{}{
		"Status": "execute",
	})

	done := make(chan chanDone, len(info.ListFiles))

	for dirName := range info.ListFiles {
		if len(info.ListFiles[dirName]) == 0 {
			continue
		}

		fmt.Printf("//запуск фильтрации для каждой директории %v\n", dirName)

		newChanStop := make(chan struct{})

		if err := sma.AddNewChanStopProcessionFiltrationTask(clientID, taskID, newChanStop); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		//запуск фильтрации для каждой директории
		go executeFiltration(done, info, sma, np, dirName, patternScript, saveMessageApp, newChanStop)
	}

	_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
		TypeMessage: "info",
		Description: fmt.Sprintf("start of a task to filter with the ID %v", taskID),
		FuncName:    fn,
	})

	fmt.Println("	//обработка информации о завершении фильтрации для каждой директории")

	//обработка информации о завершении фильтрации для каждой директории
	filteringComplete(sma, np, saveMessageApp, done)
}

//выполнение фильтрации
func executeFiltration(
	done chan<- chanDone,
	ft *configure.FiltrationTasks,
	sma *configure.StoreMemoryApplication,
	np common.NotifyParameters,
	dirName, ps string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	chanStop <-chan struct{}) {

	fmt.Printf("func 'executeFiltration' START, dir name: %v\n", dirName)

	fn := "executeFiltration"

DONE:
	for _, file := range ft.ListFiles[dirName] {
		patternScript := ps

		select {
		//выполнится если в канал придет запрос на останов фильтрации
		case <-chanStop:

			fmt.Println("func 'executeFiltration' ----- STOP -------")

			break DONE

		default:
			var successfulFiltering bool

			//заменяем путь и имя до фильтруемого файла
			patternScript = strings.Replace(patternScript, "$path_file_name", path.Join(dirName, file), -1)
			//заменяем имя файла в который будет сохранятся результат фильтрации
			patternScript = strings.Replace(patternScript, "$file_name_result", file, -1)

			//запускаем сформированный скрипт для поиска файлов
			if err := exec.Command("sh", "-c", patternScript).Run(); err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprintf("%v\t%v, file: %v", err, dirName, file),
					FuncName:    fn,
				})

				//если ошибка увеличиваем количество обработанных с ошибкой файлов
				if _, err := sma.IncrementNumNotFoundIndexFiles(np.ClientID, np.TaskID); err != nil {
					_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})
				}
			}

			//увеличиваем кол-во обработанных файлов
			if _, err := sma.IncrementNumProcessedFiles(np.ClientID, np.TaskID); err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprintf("%v\t%v, file: %v", err, dirName, file),
					FuncName:    fn,
				})
			}

			pathToFile := path.Join(ft.FileStorageDirectory, file)

			fileSize, fileHex, err := getFileParameters(pathToFile)
			if err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}

			//если файл имеет размер больше 24 байта прибавляем его к найденным и складываем общий размер найденных файлов
			if fileSize > int64(24) {
				successfulFiltering = true

				if _, err := sma.IncrementNumFoundFiles(np.ClientID, np.TaskID, fileSize); err != nil {
					_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})
				}
			} else {
				if err := os.Remove(pathToFile); err != nil {
					_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})
				}
			}

			taskInfo, err := sma.GetInfoTaskFiltration(np.ClientID, np.TaskID)
			if err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})

				fmt.Printf("func 'executeFiltration', ERROR:%v\n", err)

				break DONE
			}

			msgRes := configure.MsgTypeFiltration{
				MsgType: "filtration",
				Info: configure.DetailInfoMsgFiltration{
					TaskID:                          np.TaskID,
					TaskStatus:                      "execute",
					NumberFilesMeetFilterParameters: taskInfo.NumberFilesMeetFilterParameters,
					NumberProcessedFiles:            taskInfo.NumberProcessedFiles,
					NumberFilesFoundResultFiltering: taskInfo.NumberFilesFoundResultFiltering,
					NumberDirectoryFiltartion:       len(taskInfo.ListFiles),
					NumberErrorProcessedFiles:       taskInfo.NumberErrorProcessedFiles,
					SizeFilesMeetFilterParameters:   taskInfo.SizeFilesMeetFilterParameters,
					SizeFilesFoundResultFiltering:   taskInfo.SizeFilesFoundResultFiltering,
					PathStorageSource:               taskInfo.FileStorageDirectory,
				},
			}

			if successfulFiltering {
				msgRes.Info.FoundFilesInformation = map[string]*configure.InputFilesInformation{
					file: &configure.InputFilesInformation{
						Size: fileSize,
						Hex:  fileHex,
					},
				}
			}

			fmt.Printf("отправлена информация о задаче с ID '%v', статус задачи - 'execute', task info: '%v'\n", np.TaskID, msgRes.Info)

			resJSON, err := json.Marshal(msgRes)
			if err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})

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
		TypeProcessing: "complete",
	}

	ti, err := sma.GetInfoTaskFiltration(np.ClientID, np.TaskID)
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
		done <- sendInChan

		return
	}

	fmt.Printf("func 'executeFiltration', STOP EXECUTE DiR NAME:%v\n", dirName)

	sendInChan.TypeProcessing = ti.Status
	done <- sendInChan
}

//завершение выполнения фильтрации
func filteringComplete(
	sma *configure.StoreMemoryApplication,
	np common.NotifyParameters,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	done chan chanDone) {

	fmt.Println("func 'filteringComplete' START...")

	defer close(done)

	var dirComplete int
	var responseDone chanDone
	fn := "filteringComplete"

	taskInfo, err := sma.GetInfoTaskFiltration(np.ClientID, np.TaskID)
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	num := len(taskInfo.ListFiles)

	for dirComplete < num {
		responseDone = <-done

		fmt.Printf("func 'filteringComplete' dirComplete = %v\n", dirComplete)

		if np.TaskID == responseDone.TaskID {
			dirComplete++
		}
	}

	tp := responseDone.TypeProcessing
	if tp == "execute" {
		tp = "complete"
	}

	if err := sma.SetInfoTaskFiltration(np.ClientID, np.TaskID, map[string]interface{}{
		"Status": tp,
	}); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	fmt.Println("func 'filteringComplete' SEND MSG...")

	//отправляем сообщение о завершении фильтрации и передаем СПИСОК ВСЕХ найденных в результате фильтрации файлов
	if err := SendMessageFiltrationComplete(np.ChanRes, sma, np.ClientID, np.TaskID); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}
}
