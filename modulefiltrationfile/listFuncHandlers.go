package modulefiltrationfile

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
	"path"
	"strconv"
	"strings"
	"time"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

func createDirectoryForFiltering(sma *configure.StoreMemoryApplication, clientID, taskID, dspf string) error {
	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	dateTimeStart := time.Unix(int64(info.DateTimeStart), 0)
	dateTimeStart = dateTimeStart.UTC()

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
		UseIndex                        bool `xml:"use_index"`
		NumberFilesMeetFilterParameters int  `xml:"number_files_meet_filter_parameters"`
		NumberProcessedFiles            int  `xml:"number_processed_files"`
		NumberErrorProcessedFiles       int  `xml:"number_error_processed_files"`
	}

	task, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	i := Information{
		UseIndex:                        task.UseIndex,
		DateTimeCreateTask:              time.Now().String(),
		NumberFilesMeetFilterParameters: task.NumberFilesMeetFilterParameters,
		NumberProcessedFiles:            task.NumberProcessedFiles,
		NumberErrorProcessedFiles:       task.NumberErrorProcessedFiles,
		FilterSettings: FilterSettings{
			Protocol:      task.Protocol,
			DateTimeStart: time.Unix(int64(task.DateTimeStart), 0).UTC().String(),
			DateTimeEnd:   time.Unix(int64(task.DateTimeEnd), 0).UTC().String(),
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

	fn := "getListFilesForFiltering"

	currentTask, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
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
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(resultFoundFile.ErrMsg),
				FuncName:    fn,
			})
		}

		if resultFoundFile.Files != nil {
			if _, err := sma.AddFileToListFilesFiltrationTask(clientID, taskID, map[string][]string{resultFoundFile.Path: resultFoundFile.Files}); err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(resultFoundFile.ErrMsg),
					FuncName:    fn,
				})
			}
		}

		fullCountFiles += resultFoundFile.CountFiles
		fullSizeFiles += resultFoundFile.SizeFiles
		filesNotFound += resultFoundFile.CountFilesNotFound
		count--
	}

	if err := sma.SetInfoTaskFiltration(clientID, taskID, map[string]interface{}{
		"NumberFilesMeetFilterParameters": fullCountFiles,
		"SizeFilesMeetFilterParameters":   fullSizeFiles,
		"NumberErrorProcessedFiles":       filesNotFound,
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

	listTypeArea := map[string]string{
		"ip":    "",
		"pppoe": "(pppoes && ip) && ",
	}

	rangeFunc := func(s []string, pattern, typeElem string) string {
		countAny := len(s)
		if countAny == 0 {
			return ""
		}

		num := 0
		for _, v := range s {
			pEnd := " ||"
			if num == countAny-1 {
				if typeElem == "port" {
					pEnd = ")"
				} else {
					pEnd = "))"
				}
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

		return fmt.Sprintf("%v &&", proto)

	}

	formingPatterns := func(p *configure.FiltrationControlIPorNetorPortParameters, a, s, d, typeElem string) string {
		numAny := len(p.Any)
		numSrc := len(p.Src)
		numDst := len(p.Dst)

		var pOr, pAnd string

		if (numAny == 0) && (numSrc == 0) && (numDst == 0) {
			return ""
		}

		pAny := rangeFunc(p.Any, a, typeElem)
		pSrc := rangeFunc(p.Src, s, typeElem)
		pDst := rangeFunc(p.Dst, d, typeElem)

		if (numAny > 0) && ((numSrc > 0) || (numDst > 0)) {
			pOr = " || "
		}

		if (numSrc > 0) && (numDst > 0) {
			pAnd = " && "

			return fmt.Sprintf("(%v%v(%v%v%v))", pAny, pOr, pSrc, pAnd, pDst)
		}

		return fmt.Sprintf("(%v%v%v%v%v)", pAny, pOr, pSrc, pAnd, pDst)
	}

	patternIPAddress := func(ip *configure.FiltrationControlIPorNetorPortParameters, pProto string) string {
		h := fmt.Sprintf("(%v (host", pProto)
		s := fmt.Sprintf("(%v (src", pProto)
		d := fmt.Sprintf("(%v (dst", pProto)

		return formingPatterns(ip, h, s, d, "ip")
	}

	patternPort := func(port *configure.FiltrationControlIPorNetorPortParameters) string {
		return formingPatterns(port, "(port", "(src port", "(dst port", "port")
	}

	patternNetwork := func(network *configure.FiltrationControlIPorNetorPortParameters, pProto string) string {
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

		h := fmt.Sprintf("(%v (net", pProto)
		s := fmt.Sprintf("(%v (src net", pProto)
		d := fmt.Sprintf("(%v (dst net", pProto)

		return formingPatterns(&configure.FiltrationControlIPorNetorPortParameters{
			Any: forEachFunc(network.Any),
			Src: forEachFunc(network.Src),
			Dst: forEachFunc(network.Dst),
		}, h, s, d, "network")
	}

	//формируем шаблон для фильтрации по протоколам сетевого уровня
	pProto := patternTypeProtocol(filtrationParameters.Protocol)

	//формируем шаблон для фильтрации по ip адресам
	pIP := patternIPAddress(&filtrationParameters.Filters.IP, pProto)

	//формируем шаблон для фильтрации по сетевым портам
	pPort := patternPort(&filtrationParameters.Filters.Port)

	//формируем шаблон для фильтрации по сетям
	pNetwork := patternNetwork(&filtrationParameters.Filters.Network, pProto)

	if len(pPort) > 0 && (len(pIP) > 0 || len(pNetwork) > 0) {
		pAnd = " && "
	}

	if len(pIP) > 0 && len(pNetwork) > 0 {
		patterns = fmt.Sprintf(" (%v || %v)%v%v", pIP, pNetwork, pAnd, pPort)
	} else {
		patterns = fmt.Sprintf("%v%v%v%v", pIP, pNetwork, pAnd, pPort)
	}

	return fmt.Sprintf("tcpdump -r $path_file_name '%v%v || (vlan && %v)' -w %v", listTypeArea[typeArea], patterns, patterns, path.Join(filtrationParameters.FileStorageDirectory, "$file_name_result"))
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

//GetListFoundFiles получаем список файлов полученных в результате фильтрации
func GetListFoundFiles(directoryResultFilter string) (map[string]*configure.InputFilesInformation, int64, error) {
	files, err := ioutil.ReadDir(directoryResultFilter)
	if err != nil {
		return nil, 0, err
	}

	var sizeFiles int64
	newList := map[string]*configure.InputFilesInformation{}

	for _, file := range files {
		if (file.Name() != "README.xml") && (file.Size() > 24) {
			_, hex, _ := common.GetFileParameters(path.Join(directoryResultFilter, file.Name()))

			newList[file.Name()] = &configure.InputFilesInformation{
				Size: file.Size(),
				Hex:  hex,
			}

			sizeFiles += file.Size()
		}
	}

	return newList, sizeFiles, nil
}

//SendMessageFiltrationComplete передача сообщений о завершении фильтрации
// при этом передается СПИСОК ВСЕХ найденных в результате фильтрации файлов
func SendMessageFiltrationComplete(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID string) error {

	const sizeChunk = 100

	taskInfo, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	//получить список найденных, в результате фильтрации, файлов
	listFiles, sizeFiles, err := GetListFoundFiles(taskInfo.FileStorageDirectory)
	if err != nil {
		return err
	}

	numFilesFound := len(listFiles)

	msgRes := configure.MsgTypeFiltration{
		MsgType: "filtration",
		Info: configure.DetailInfoMsgFiltration{
			TaskID:                          taskID,
			TaskStatus:                      taskInfo.Status,
			NumberFilesMeetFilterParameters: taskInfo.NumberFilesMeetFilterParameters,
			NumberProcessedFiles:            taskInfo.NumberProcessedFiles,
			NumberFilesFoundResultFiltering: numFilesFound,
			NumberDirectoryFiltartion:       len(taskInfo.ListFiles),
			NumberErrorProcessedFiles:       taskInfo.NumberErrorProcessedFiles,
			SizeFilesMeetFilterParameters:   taskInfo.SizeFilesMeetFilterParameters,
			SizeFilesFoundResultFiltering:   sizeFiles,
			PathStorageSource:               taskInfo.FileStorageDirectory,
		},
	}

	//если количество найденных файлов 0 или меньше чем константа sizeChunk
	if (numFilesFound == 0) || (numFilesFound < sizeChunk) {
		msgRes.Info.FoundFilesInformation = listFiles

		resJSON, err := json.Marshal(msgRes)
		if err != nil {
			return err
		}

		//сообщение о завершении процесса фильтрации
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resJSON,
		}

		return nil
	}

	/* отправляем множество частей одного и того же сообщения */

	//получаем количество частей сообщений
	numParts := common.CountNumberParts(int64(numFilesFound), sizeChunk)

	numberMessageParts := [2]int{0, numParts}

	//отправляются последующие части сообщений содержащие списки имен файлов
	for i := 1; i <= numParts; i++ {
		chunkListFiles := common.GetChunkListFilesFound(listFiles, i, numParts, sizeChunk)

		numberMessageParts[0] = i
		msgRes.Info.NumberMessagesParts = numberMessageParts
		msgRes.Info.FoundFilesInformation = chunkListFiles

		resJSON, err := json.Marshal(msgRes)
		if err != nil {
			return err
		}

		//сообщение о завершении процесса фильтрации
		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resJSON,
		}
	}

	return nil
}

//sendMsgTypeFilteringRefused отправить сообщение для удаления информации о задаче на ISEMS-NIH_master
func sendMsgTypeFilteringRefused(cwt chan<- configure.MsgWsTransmission, clientID, taskID string) error {
	resJSON, err := json.Marshal(configure.MsgTypeFiltration{
		MsgType: "filtration",
		Info: configure.DetailInfoMsgFiltration{
			TaskID:     taskID,
			TaskStatus: "refused",
		},
	})
	if err != nil {
		return err
	}

	//сообщение о ходе процесса фильтрации
	cwt <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &resJSON,
	}

	return nil
}
