package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
*
* Версия 0.1, дата релиза 16.05.2019
* */

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//ProcessingFiltration выполняет фильтрацию сет. трафика
func ProcessingFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID, rootDirStoringFiles string) {

	fmt.Println("START function 'ProcessingFiltration'...")

	saveMessageApp := savemessageapp.New()
	errMsg := configure.MsgTypeError{
		MsgType: "error",
		Info: configure.DetailInfoMsgError{
			TaskID: taskID,
		},
	}

	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		errMsg.Info.ErrorName = "invalid value received"
		errMsg.Info.ErrorDescription = "Невозможно начать выполнение фильтрации, принят некорректный идентификатор клиента или задачи"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

		return
	}

	fmt.Printf("Параметры задачи по фильтрации:\n%v\n", info)

	//создаем директорию для хранения отфильтрованных файлов
	if err := createDirectoryForFiltering(sma, clientID, taskID, rootDirStoringFiles); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		errMsg.Info.ErrorName = "unable to create directory"
		errMsg.Info.ErrorDescription = "Ошибка при создании директории для хранения отфильтрованных файлов"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		return
	}

	//создаем файл README с описанием параметров фильтрации
	if err := createFileReadme(sma, clientID, taskID); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		errMsg.Info.ErrorName = "unable to create directory"
		errMsg.Info.ErrorDescription = "Ошибка при создании директории для хранения отфильтрованных файлов"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		return
	}

	//инициализируем выполнение фильтрации
	if err := executeFiltration(cwtResText, sma, clientID, taskID); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		errMsg.Info.ErrorName = "unable to create directory"
		errMsg.Info.ErrorDescription = "Ошибка при создании директории для хранения отфильтрованных файлов"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		return
	}

	/*
		отправка сообщений в канал, для дальнейшей передачи ISEMS-NIH_master
		cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &msgJSON,
				}
	*/
}

func createDirectoryForFiltering(sma *configure.StoreMemoryApplication, clientID, taskID, dspf string) error {
	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	dateTimeStart := time.Unix(int64(info.DateTimeStart), 0)

	dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + taskID
	filePath := path.Join(dspf, "/", dirName)

	if err := os.MkdirAll(filePath, 0766); err != nil {
		return err
	}

	if err := sma.SetInfoTaskFiltration(
		clientID,
		taskID,
		map[string]interface{}{
			"FileStorageDirectory": filePath,
		}); err != nil {
		return err
	}

	return nil
}

func createFileReadme(sma *configure.StoreMemoryApplication, clientID, taskID string) error {
	type FiltrationControlIPorNetorPortParameters struct {
		Any []string `xml:"any>value"`
		Src []string `xml:"src>value"`
		Dst []string `xml:"dst>value"`
	}

	type FilterSettings struct {
		Protocol      string                                   `xml:"filters>protocol"`
		DateTimeStart string                                   `xml:"filters>date_time_start"`
		DateTimeEnd   string                                   `xml:"filters>date_time_end"`
		IP            FiltrationControlIPorNetorPortParameters `xml:"filters>ip"`
		Port          FiltrationControlIPorNetorPortParameters `xml:"filters>port"`
		Network       FiltrationControlIPorNetorPortParameters `xml:"filters>network"`
	}

	type Information struct {
		XMLName            xml.Name `xml:"information"`
		DateTimeCreateTask string   `xml:"date_time_create_task"`
		FilterSettings
		UseIndex                bool `xml:"use_index"`
		CountIndexFiles         int  `xml:"count_index_files"`
		CountProcessedFiles     int  `xml:"count_processed_files"`
		CountNotFoundIndexFiles int  `xml:"count_not_found_index_files"`
	}

	task, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	tct := time.Unix(int64(time.Now().Unix()), 0)
	dtct := strconv.Itoa(tct.Day()) + " " + tct.Month().String() + " " + strconv.Itoa(tct.Year()) + " " + strconv.Itoa(tct.Hour()) + ":" + strconv.Itoa(tct.Minute())

	i := Information{
		UseIndex:                task.UseIndex,
		DateTimeCreateTask:      dtct,
		CountIndexFiles:         task.CountIndexFiles,
		CountProcessedFiles:     task.CountProcessedFiles,
		CountNotFoundIndexFiles: task.CountNotFoundIndexFiles,
		FilterSettings: FilterSettings{
			Protocol:      task.Protocol,
			DateTimeStart: fmt.Sprint(time.Unix(int64(task.DateTimeStart), 0)),
			DateTimeEnd:   fmt.Sprint(time.Unix(int64(task.DateTimeEnd), 0)),
			IP: FiltrationControlIPorNetorPortParameters{
				Any: task.Filters.IP.Any,
				Src: task.Filters.IP.Src,
				Dst: task.Filters.IP.Dst,
			},
			Port: FiltrationControlIPorNetorPortParameters{
				Any: task.Filters.Port.Any,
				Src: task.Filters.Port.Src,
				Dst: task.Filters.Port.Dst,
			},
			Network: FiltrationControlIPorNetorPortParameters{
				Any: task.Filters.Network.Any,
				Src: task.Filters.Network.Src,
				Dst: task.Filters.Network.Dst,
			},
		},
	}

	output, err := xml.MarshalIndent(i, "  ", "    ")
	if err != nil {
		return err
	}

	fmt.Printf("storageDir: %v, README.xml", task.FileStorageDirectory)

	f, err := os.OpenFile(path.Join(task.FileStorageDirectory, "README.xml"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(output); err != nil {
		return err
	}

	return nil
}

//currentListFilesFilteration содержит информацию о найденных файлах
type currentListFilesFilteration struct {
	Path               string
	Files              []string
	CountFiles         int
	CountFilesNotFound int
	SizeFiles          int64
	ErrMsg             error
}

func searchFiles(result chan<- currentListFilesFilteration, disk string, currentTask *configure.FiltrationTasks) {
	clff := currentListFilesFilteration{Path: disk}

	if currentTask.UseIndex {
		for _, file := range currentTask.ListFiles[disk] {
			fileInfo, err := os.Stat(path.Join(disk, file))
			if err != nil {
				clff.CountFilesNotFound++

				continue
			}

			clff.Files = append(clff.Files, file)
			clff.SizeFiles += fileInfo.Size()
			clff.CountFiles++
		}

		result <- clff

		return
	}

	files, err := ioutil.ReadDir(disk)
	if err != nil {
		clff.ErrMsg = err
		result <- clff

		return
	}

	for _, file := range files {
		fileIsUnixDate := file.ModTime().Unix()

		if (uint64(fileIsUnixDate) > currentTask.DateTimeStart) && (uint64(fileIsUnixDate) < currentTask.DateTimeEnd) {
			clff.Files = append(clff.Files, file.Name())
			clff.SizeFiles += file.Size()
			clff.CountFiles++
		}
	}

	result <- clff
}

//getListFilesForFiltering формирует и сохранаяет список файлов удовлетворяющих временному диапазону
// или найденных при поиске по индексам
func getListFilesForFiltering(sma *configure.StoreMemoryApplication, clientID, taskID string) error {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	currentTask, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return err
	}

	as := sma.GetApplicationSetting()
	ld := as.StorageFolders

	var fullCountFiles, filesNotFound int
	var fullSizeFiles int64

	var result = make(chan currentListFilesFilteration, len(ld))
	defer close(result)

	var count int
	if currentTask.UseIndex {
		count = len(currentTask.ListFiles)
		for disk := range currentTask.ListFiles {
			go searchFiles(result, disk, currentTask)
		}
	} else {
		count = len(ld)
		for _, disk := range ld {
			go searchFiles(result, disk, currentTask)
		}
	}

	for count > 0 {
		resultFoundFile := <-result

		if resultFoundFile.ErrMsg != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(resultFoundFile.ErrMsg))
		}

		if resultFoundFile.Files != nil {
			if _, err := sma.AddFileToListFilesFiltrationTask(clientID, taskID, map[string][]string{resultFoundFile.Path: resultFoundFile.Files}); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(resultFoundFile.ErrMsg))
			}
		}

		fullCountFiles += resultFoundFile.CountFiles
		fullSizeFiles += resultFoundFile.SizeFiles
		filesNotFound += resultFoundFile.CountFilesNotFound
		count--
	}

	if err := sma.SetInfoTaskFiltration(clientID, taskID, map[string]interface{}{
		"CountIndexFiles":         fullCountFiles,
		"SizeIndexFiles":          fullSizeFiles,
		"CountNotFoundIndexFiles": filesNotFound,
	}); err != nil {
		return err
	}

	return nil
}

//выполнение фильтрации
func executeFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID string) error {

	if err := getListFilesForFiltering(sma, clientID, taskID); err != nil {
		return err
	}

	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	//проверяем количество файлов которые не были найдены при поиске их по индексам
	if info.CountNotFoundIndexFiles > 0 {

		//ОТПРАВИТЬ ИНФОРМАЦИИОННОЕ СООБЩЕНИЕ О НЕ ВСЕХ ФАЙЛАХ найденных по индексам
		infoMsg := configure.MsgTypeInformation{
			MsgType: "error",
			Info: configure.InformationMessage{
				TaskID: taskID,
				/*
					Дописать тип информационного сообщения
				*/
			},
		}

		/*cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}*/
	}

	return nil

	/*
	   !!! ВНИМАНИЕ !!!
	   Сделать создание списка файлов удовлетворяющих заданным условиям
	   И оттестировать эту функцию
	*/
}
