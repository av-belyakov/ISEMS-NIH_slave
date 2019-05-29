package modulefiltrationfile

import (
	"crypto/md5"
	"encoding/hex"
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

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

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
		UseIndex                        bool `xml:"use_index"`
		NumberFilesMeetFilterParameters int  `xml:"number_files_meet_filter_parameters"`
		NumberProcessedFiles            int  `xml:"number_processed_files"`
		NumberErrorProcessedFiles       int  `xml:"number_error_processed_files"`
	}

	task, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	tct := time.Unix(int64(time.Now().Unix()), 0)
	dtct := strconv.Itoa(tct.Day()) + " " + tct.Month().String() + " " + strconv.Itoa(tct.Year()) + " " + strconv.Itoa(tct.Hour()) + ":" + strconv.Itoa(tct.Minute())

	i := Information{
		UseIndex:                        task.UseIndex,
		DateTimeCreateTask:              dtct,
		NumberFilesMeetFilterParameters: task.NumberFilesMeetFilterParameters,
		NumberProcessedFiles:            task.NumberProcessedFiles,
		NumberErrorProcessedFiles:       task.NumberErrorProcessedFiles,
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

//sendMessageFiltrationStop передача сообщений о завершении фильтрации
// при этом передается СПИСОК ВСЕХ найденных в результате фильтрации файлов
func sendMessageFiltrationStop(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID string) {
	/*
		msgRes := configure.MsgTypeFiltration{
			MsgType: "filtration",
			Info: configure.DetailInfoMsgFiltration{
				TaskID:                          mp.TaskID,
				TaskStatus:                      "execute",
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
		mp.ChanRes <- configure.MsgWsTransmission{
			ClientID: mp.ClientID,
			Data:     &resJSON,
		}
	*/
}
