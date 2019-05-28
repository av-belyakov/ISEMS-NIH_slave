package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
*
* Версия 0.3, дата релиза 27.05.2019
* */

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

type msgParameters struct {
	ClientID, TaskID string
	ChanRes          chan<- configure.MsgWsTransmission
}

//ChanDone содержит информацию о завершенной задаче
type chanDone struct {
	ClientID, TaskID, DirectoryName, TypeProcessing string
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

	as := sma.GetApplicationSetting()

	//формируем шаблон фильтрации
	patternScript := createPatternScript(info, as.TypeAreaNetwork)

	done := make(chan chanDone, len(info.ListFiles))

	for dirName := range info.ListFiles {
		if len(info.ListFiles[dirName]) == 0 {
			continue
		}

		//запуск фильтрации для каждой директории
		go executeFiltration(done, info, sma, clientID, taskID, dirName, patternScript)
	}

	/*formingMessageFilterComplete := FormingMessageFilterComplete{
		taskIndex:      taskIndex,
		remoteIP:       prf.RemoteIP,
		countDirectory: infoTaskFilter.CountDirectoryFiltering,
	}*/

	_ = saveMessageApp.LogMessage("info", fmt.Sprintf("start of a task to filter with the ID %v", taskID))

	//обработка информации о завершении фильтрации для каждой директории
	go filteringComplete(done) //, &formingMessageFilterComplete, prf, ift)
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

//createPatternScript подготовка шаблона для фильтрации
// Логика работы: сетевой протокол '&&' (набор IP '||' набор Network) '&&' набор портов,
// (ANY '||' (SRC '&&' DST))
func createPatternScript(filtrationParameters *configure.FiltrationTasks, typeArea string) string {
	var pAnd, patterns string

	rangeFunc := func(s []string, pattern string) string {
		countAny := len(s)
		if countAny == 0 {
			return ""
		}

		num := 0
		for _, v := range s {
			pEnd := " ||"
			if num == countAny-1 {
				pEnd = ")"
			}

			pattern += " " + v + pEnd
			num++
		}

		return pattern
	}

	patternTypeProtocol := func(proto string) string {
		if proto == "any" || proto == "" {
			return ""
		}

		return fmt.Sprintf("((ip && %v) || (vlan && %v)) && ", proto, proto)

	}

	formingPatterns := func(p *configure.FiltrationControlIPorNetorPortParameters, a, s, d string) string {
		numAny := len(p.Any)
		numSrc := len(p.Src)
		numDst := len(p.Dst)

		var pOr, pAnd string

		if (numAny == 0) && (numSrc == 0) && (numDst == 0) {
			return ""
		}

		pAny := rangeFunc(p.Any, a)
		pSrc := rangeFunc(p.Src, s)
		pDst := rangeFunc(p.Dst, d)

		if (numAny > 0) && ((numSrc > 0) || (numDst > 0)) {
			pOr = " || "
		}

		if (numSrc > 0) && (numDst > 0) {
			pAnd = " && "

			return fmt.Sprintf("(%v%v(%v%v%v))", pAny, pOr, pSrc, pAnd, pDst)
		}

		return fmt.Sprintf("(%v%v%v%v%v)", pAny, pOr, pSrc, pAnd, pDst)
	}

	patternIPAddress := func(ip *configure.FiltrationControlIPorNetorPortParameters) string {
		return formingPatterns(ip, "(host", "(src", "(dst")
	}

	patternPort := func(port *configure.FiltrationControlIPorNetorPortParameters) string {
		return formingPatterns(port, "(port", "(src port", "(dst port")
	}

	patternNetwork := func(network *configure.FiltrationControlIPorNetorPortParameters) string {
		//приводим значение сетей к валидным сетевым маскам
		forEachFunc := func(list []string) []string {
			newList := make([]string, 0, len(list))

			for _, v := range list {
				t := strings.Split(v, "/")

				mask, _ := strconv.Atoi(t[1])

				ipv4Addr := net.ParseIP(t[0])
				ipv4Mask := net.CIDRMask(mask, 32)

				newList = append(newList, fmt.Sprintf("%v/%v", ipv4Addr.Mask(ipv4Mask).String(), t[1]))
			}

			return newList
		}

		return formingPatterns(&configure.FiltrationControlIPorNetorPortParameters{
			Any: forEachFunc(network.Any),
			Src: forEachFunc(network.Src),
			Dst: forEachFunc(network.Dst),
		}, "(net", "(src net", "(dst net")
	}

	//формируем шаблон для фильтрации по протоколам сетевого уровня
	pProto := patternTypeProtocol(filtrationParameters.Protocol)

	//формируем шаблон для фильтрации по ip адресам
	pIP := patternIPAddress(&filtrationParameters.Filters.IP)

	//формируем шаблон для фильтрации по сетевым портам
	pPort := patternPort(&filtrationParameters.Filters.Port)

	//формируем шаблон для фильтрации по сетям
	pNetwork := patternNetwork(&filtrationParameters.Filters.Network)

	if len(pPort) > 0 && (len(pIP) > 0 || len(pNetwork) > 0) {
		pAnd = " && "
	}

	if len(pIP) > 0 && len(pNetwork) > 0 {
		patterns = fmt.Sprintf(" (%v || %v)%v%v", pIP, pNetwork, pAnd, pPort)
	} else {
		patterns = fmt.Sprintf("%v%v%v%v", pIP, pNetwork, pAnd, pPort)
	}

	return fmt.Sprintf("tcpdump -r $path_file_name '%v%v%v || (vlan && %v)' -w %v", typeArea, pProto, patterns, patterns, filtrationParameters.FileStorageDirectory)
}

func getFileParameters(filePath string) (int64, string, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		return 0, "", err
	}
	defer fd.Close()

	fileInfo, err := fd.Stat()
	if err != nil {
		return 0, "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, fd); err != nil {
		return 0, "", err
	}

	return fileInfo.Size(), hex.EncodeToString(h.Sum(nil)), nil
}

//выполнение фильтрации
func executeFiltration(done chan<- chanDone, ft *configure.FiltrationTasks, sma *configure.StoreMemoryApplication, clientID, taskID, dirName, patternScript string) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	var statusProcessedFile bool

DONE:
	for _, file := range ft.ListFiles[dirName] {
		select {
		//выполнится если в канал придет запрос на останов фильтрации
		case <-ft.ChanStopFiltration:
			break DONE

		default:
			pathAndFileName := path.Join(dirName, file)

			patternScript = strings.Replace(patternScript, "$path_file_name", pathAndFileName, -1)

			if err := exec.Command("sh", "-c", patternScript).Run(); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprintf("%v\t%v, file: %v\n", err, dirName, file))

				//если ошибка увеличиваем количество обработанных с ошибкой файлов
				if _, err := sma.IncrementNumNotFoundIndexFiles(clientID, taskID); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

				statusProcessedFile = false
			} else {
				statusProcessedFile = true
			}

			//увеличиваем кол-во обработанных файлов
			if _, err := sma.IncrementNumProcessedFiles(clientID, taskID); err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprintf("%v\t%v, file: %v\n", err, dirName, file))
			}

			//если файл имеет размер больше 24 байта прибавляем его к найденным и складываем общий размер найденных файлов
			fileSize, fileHex, err := getFileParameters(pathAndFileName)
			if err != nil {
				_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
			}
			if fileSize > int64(24) {
				if _, err := sma.IncrementNumFoundFiles(clientID, taskID, fileSize); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
			}

			//формируем канал для передачи информации о фильтрации
			prf.AccessClientsConfigure.ChanInfoFilterTask <- configure.ChanInfoFilterTask{
				TaskIndex:           ppf.TaskIndex,
				RemoteIP:            prf.RemoteIP,
				TypeProcessing:      "execute",
				DirectoryName:       ppf.DirectoryName,
				ProcessingFileName:  file,
				CountFilesFound:     countFiles,
				CountFoundFilesSize: fullSizeFiles,
				StatusProcessedFile: statusProcessedFile,
			}
		}
	}

	done <- chanDone{
		ClientID:       clientID,
		TaskID:         taskID,
		DirectoryName:  dirName,
		TypeProcessing: "complete",
	}
}

//завершение выполнения фильтрации
func filteringComplete(done chan chanDone) {

}
