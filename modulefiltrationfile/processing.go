package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
*
* Версия 0.3, дата релиза 27.05.2019
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

type msgParameters struct {
	ClientID, TaskID string
	ChanRes          chan<- configure.MsgWsTransmission
}

func (mp *msgParameters) sendErrMsg(errName, errDesc string) error {
	msg := configure.MsgTypeError{
		MsgType: "error",
		Info: configure.DetailInfoMsgError{
			TaskID:           mp.TaskID,
			ErrorName:        errName,
			ErrorDescription: errDesc,
		},
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	mp.ChanRes <- configure.MsgWsTransmission{
		ClientID: mp.ClientID,
		Data:     &msgJSON,
	}

	return nil
}

func (mp *msgParameters) sendMsgNotify(notifyType, notifyDesc, typeActionPerform string) error {
	msg := configure.MsgTypeNotification{
		MsgType: "notification",
		Info: configure.DetailInfoMsgNotification{
			TaskID:              mp.TaskID,
			Section:             "filtration control",
			TypeActionPerformed: typeActionPerform,
			CriticalityMessage:  notifyType,
			Description:         notifyDesc,
		},
	}

	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	mp.ChanRes <- configure.MsgWsTransmission{
		ClientID: mp.ClientID,
		Data:     &msgJSON,
	}

	return nil
}

//ProcessingFiltration выполняет фильтрацию сет. трафика
func ProcessingFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID, rootDirStoringFiles string) {

	fmt.Println("START function 'ProcessingFiltration'...")

	saveMessageApp := savemessageapp.New()

	mp := msgParameters{
		ClientID: clientID,
		TaskID:   taskID,
		ChanRes:  cwtResText,
	}

	//информируем клиента о начале создания списка файлов удовлетворяющих параметрам фильтрации
	d := "Инициализирована задача по фильтрации сетевого трафика, идет поиск файлов удовлетворяющих параметрам фильтрации"
	if err := mp.sendMsgNotify("info", d, "start"); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//строим список файлов удовлетворяющих параметрам фильтрации
	if err := getListFilesForFiltering(sma, clientID, taskID); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		en := "cannot create a list of files"
		ed := "Ошибка, невозможно создать список файлов удовлетворяющий параметрам фильтрации"

		if err := mp.sendErrMsg(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

		en := "invalid value received"
		ed := "Невозможно начать выполнение фильтрации, принят некорректный идентификатор клиента или задачи"

		if err := mp.sendErrMsg(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	//проверяем количество файлов которые не были найдены при поиске их по индексам
	if info.CountNotFoundIndexFiles > 0 {
		d := "Внимание, фильтрация выполняется по файлам полученным при поиске по индексам. Однако, на диске были найдены не все файлы перечисленные в индексах"
		if err := mp.sendMsgNotify("warning", d, "start"); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}
	}

	//поверяем количество файлов по которым необходимо выполнить фильтрацию
	if info.CountIndexFiles == 0 {
		en := "no files matching configured interval"
		ed := "Внимание, фильтрация остановлена так как не найдены файлы удовлетворяющие параметрам фильтрации"
		if err := mp.sendErrMsg(en, ed); err != nil {
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

		if err := mp.sendErrMsg(en, ed); err != nil {
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

		if err := mp.sendErrMsg(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	//формируем шаблон фильтрации
	patternScript, err := createPatternScript(info)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		en := "it is impossible to form a pattern of filter parameters"
		ed := "Невозможно сформировать шаблон из параметров фильтрации"

		if err := mp.sendErrMsg(en, ed); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		return
	}

	/*

	   Будем передовать список, найденных в результате фильтрации файлов,
	   при завершении задачи по фильтрации (статус задачи 'complete' или 'stop')

	   НЕ ОБРАЩАТЬ ВНИМАНИЕ что moth_go отправляет в начале, они все равно не использутся
	   видимо у меня были какие т мысли по поводу этого, но я так их и не реализовал

	*/

	//запуск фильтрации для каждой директории
	executeFiltration(info, patternScript)

	//обработка информации о завершении фильтрации для каждой директории
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

		if (int64(fileIsUnixDate) > currentTask.DateTimeStart) && (int64(fileIsUnixDate) < currentTask.DateTimeEnd) {
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

func createPatternScript(filtrationParameters *configure.FiltrationTasks) (string, error) {
	//err := it is impossible to form a pattern of filter parameters
	var pattern string

	return pattern, nil
}

/*
tcpdump -r 1496732553_2017_06_06____10_02_33_992898.tdp
'((host 37.147.110.67 || 31.13.21.122) || ((src 42.118.143.87 || 2.1.1.2) && (dst 80.245.123.60 || 3.1.1.2)))
|| (vlan && ((host 37.147.110.67 || 31.13.21.122) && (src 42.118.143.87 || 2.1.1.2) && (dst 80.245.123.60 || 3.1.1.2)))'


tcpdump -r 1496732553_2017_06_06____10_02_33_992898.tdp
'((host 37.147.110.67 || 31.13.21.122) && (src 42.118.143.87 || 2.1.1.2) && (dst 80.245.123.60 || 3.1.1.2))
&& ((port 80 || 45) && (dport 80)) || (vlan && ((host 37.147.110.67 || 31.13.21.122)
&& (src 42.118.143.87 || 2.1.1.2) && (dst 80.245.123.60 || 3.1.1.2))))'
*/

//формируем шаблон для фильтрации
/*func patternBashScript(ppf PatternParametersFiltering, mtf *configure.MessageTypeFilter) string {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	getPatternNetwork := func(network string) (string, error) {
		networkTmp := strings.Split(network, "/")
		if len(networkTmp) < 2 {
			return "", errors.New("incorrect network mask value")
		}

		maskInt, err := strconv.ParseInt(networkTmp[1], 10, 64)
		if err != nil {
			return "", err
		}

		if maskInt < 0 || maskInt > 32 {
			return "", errors.New("the value of 'mask' should be in the range from 0 to 32")
		}

		ipv4Addr := net.ParseIP(networkTmp[0])
		ipv4Mask := net.CIDRMask(24, 32)
		newNetwork := ipv4Addr.Mask(ipv4Mask).String()

		return newNetwork + "/" + networkTmp[1], nil
	}

	getIPAddressString := func(ipaddreses []string) (searchHosts string) {
		if len(ipaddreses) != 0 {
			if len(ipaddreses) == 1 {
				searchHosts = "'(host " + ipaddreses[0] + " || (vlan && host " + ipaddreses[0] + "))'"
			} else {
				var hosts string
				for key, ip := range ipaddreses {
					if key == 0 {
						hosts += "(host " + ip
					} else if key == (len(ipaddreses) - 1) {
						hosts += " || " + ip + ")"
					} else {
						hosts += " || " + ip
					}
				}

				searchHosts = "'" + hosts + " || (vlan && " + hosts + ")'"
			}
		}
		return searchHosts
	}

	getNetworksString := func(networks []string) (searchNetworks string) {
		if len(networks) != 0 {
			if len(networks) == 1 {
				networkPattern, err := getPatternNetwork(networks[0])
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

				searchNetworks += "'(net " + networkPattern + " || (vlan && net " + networkPattern + "))'"
			} else {
				var network string
				for key, net := range networks {
					networkPattern, err := getPatternNetwork(net)
					if err != nil {
						_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
					}

					if key == 0 {
						network += "(net " + networkPattern
					} else if key == (len(networks) - 1) {
						network += " || " + networkPattern + ")"
					} else {
						network += " || " + networkPattern
					}
				}
				searchNetworks = "'" + network + " || (vlan && " + network + ")'"
			}
		}
		return searchNetworks
	}

	bind := " "
	//формируем строку для поиска хостов
	searchHosts := getIPAddressString(mtf.Info.Settings.IPAddress)

	//формируем строку для поиска сетей
	searchNetwork := getNetworksString(mtf.Info.Settings.Network)

	if len(mtf.Info.Settings.IPAddress) > 0 && len(mtf.Info.Settings.Network) > 0 {
		bind = " or "
	}

	listTypeArea := map[int]string{
		1: " ",
		2: " '(pppoes && ip)' and ",
	}

	pattern := " tcpdump -r " + ppf.DirectoryName + "/$files"
	pattern += listTypeArea[ppf.TypeAreaNetwork] + searchHosts + bind + searchNetwork
	pattern += " -w " + ppf.PathStorageFilterFiles + "/$files;"

	return pattern
}*/

//выполнение фильтрации
func executeFiltration(ft *configure.FiltrationTasks, patternScript string) {

}
