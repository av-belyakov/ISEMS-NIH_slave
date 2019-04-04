package telemetry

/*
* Описание структур и методов сбора системной информации
* Версия 0.1, дата релиза 04.04.2019
* */

import (
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

//Memory содержит информацию об используемой ПО
type Memory struct {
	Total int `json:"total"`
	Used  int `json:"used"`
	Free  int `json:"free"`
}

//CurrentDisk начальное и конечное время для файлов сет. трафика
type CurrentDisk struct {
	DateMin int `json:"dateMin"`
	DateMax int `json:"dateMax"`
}

//DetailedInformation системная информация
type DetailedInformation struct {
	CurrentDateTime    int64                     `json:"currentDateTime"`
	DiskSpace          []map[string]string       `json:"diskSpace"`
	TimeInterval       map[string]map[string]int `json:"timeInterval"`
	RandomAccessMemory Memory                    `json:"randomAccessMemory"`
	LoadCPU            float64                   `json:"loadCPU"`
	LoadNetwork        map[string]map[string]int `json:"loadNetwork"`
}

//SysInfo полная системная информация подготовленная к отправке
type SysInfo struct {
	MessageType string              `json:"messageType"`
	Info        DetailedInformation `json:"info"`
}

//AnswerNetIface содержит информацию по сетевому интерфейсу
type AnswerNetIface struct {
	Iface  string
	Data   map[string]int
	ErrMsg error
}

//DateTimeRange min и max время найденных файлов
type DateTimeRange struct {
	DiskName string
	Min, Max int
	ErrMsg   error
}

func getDateTimeRange(result chan<- DateTimeRange, currentDisk string) {
	var array []int

	files, err := ioutil.ReadDir(currentDisk)
	if err != nil {
		result <- DateTimeRange{currentDisk, 0, 0, err}

		return
	}

	for _, file := range files {
		unixDate := file.ModTime().Unix() * 1000
		array = append(array, int(unixDate))
	}
	sort.Ints(array)

	var min, max int
	if len(array) > 0 {
		min = array[0]
		max = array[len(array)-1]

		array = []int{}
		files = []os.FileInfo{}
	}

	result <- DateTimeRange{currentDisk, min, max, nil}
}

//CreateFilesRange формирует временный интервал файлов хранящихся на дисках
func (sysInfo *SysInfo) CreateFilesRange(done chan<- struct{}, errMsg chan<- error, cDisk []string) {
	var result = make(chan DateTimeRange, len(cDisk))
	var resultDateTimeRange DateTimeRange

	for _, disk := range cDisk {
		go getDateTimeRange(result, disk)
	}

	mapTimeInterval := make(map[string]map[string]int)

	go func() {
		for i := 0; i < len(cDisk); i++ {
			resultDateTimeRange = <-result

			if resultDateTimeRange.ErrMsg != nil {
				errMsg <- resultDateTimeRange.ErrMsg
			}

			mapTimeInterval[resultDateTimeRange.DiskName] = map[string]int{
				"dateMin": resultDateTimeRange.Min,
				"dateMax": resultDateTimeRange.Max,
			}

			sysInfo.Info.TimeInterval = mapTimeInterval
		}

		close(result)

		done <- struct{}{}
	}()
}

//CreateDiskSpace формирует информация о свободном дисковом пространстве
func (sysInfo *SysInfo) CreateDiskSpace() error {
	stdout, err := exec.Command("sh", "-c", "df -hl|grep dev").Output()
	if err != nil {
		return err
	}

	arrayStringDevises := strings.Split(string(stdout), "\n")

	resultDiskSpace := make(map[string]string)

	for _, strDev := range arrayStringDevises {
		arrayString := regexp.MustCompile("\\s+").Split(strDev, -1)

		if len(arrayString) == 1 {
			continue
		}

		resultDiskSpace = map[string]string{
			"diskName": arrayString[0],
			"mounted":  arrayString[5],
			"maxSpace": arrayString[1],
			"used":     arrayString[4],
		}
		sysInfo.Info.DiskSpace = append(sysInfo.Info.DiskSpace, resultDiskSpace)
	}
	return nil
}

//CreateRandomAccessMemory формирует информация о свободной оперативной памяти
func (sysInfo *SysInfo) CreateRandomAccessMemory() error {
	stdout, err := exec.Command("sh", "-c", "free|sed -n '2p'").Output()
	if err != nil {
		return err
	}

	freeMemory := string(stdout)
	arrayString := regexp.MustCompile("\\s+").Split(freeMemory, -1)

	sysInfo.Info.RandomAccessMemory.Total, _ = strconv.Atoi(arrayString[1])
	sysInfo.Info.RandomAccessMemory.Used, _ = strconv.Atoi(arrayString[2])
	sysInfo.Info.RandomAccessMemory.Free, _ = strconv.Atoi(arrayString[3])

	return nil
}

//CreateLoadCPU формирует информация по загрузке центрального процессора
func (sysInfo *SysInfo) CreateLoadCPU() error {
	stdout, err := exec.Command("sh", "-c", "grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$4+$5)} END {print usage}'").Output()
	if err != nil {
		return err
	}

	var strCPU []byte
	if len(stdout) < 4 {
		strCPU = stdout
	} else {
		strCPU = stdout[:len(stdout)-4]
	}

	infoCPU, err := strconv.ParseFloat(string(strCPU), 64)
	if err != nil {
		return err
	}

	sysInfo.Info.LoadCPU = infoCPU

	return nil
}

func getLoadingNetworkInterface(result chan<- AnswerNetIface, iface string) {
	script := " RX1=$(cat /sys/class/net/" + iface + "/statistics/rx_bytes); "
	script += " TX1=$(cat /sys/class/net/" + iface + "/statistics/tx_bytes); "
	script += " sleep 1; "
	script += " RX2=$(cat /sys/class/net/" + iface + "/statistics/rx_bytes); "
	script += " TX2=$(cat /sys/class/net/" + iface + "/statistics/tx_bytes); "
	script += " echo \"$(( (${TX2} - ${TX1}) / 1024 * 8 ))\"; "
	script += " echo \"$(( (${RX2} - ${RX1}) / 1024 * 8 ))\"; "

	stdout, err := exec.Command("sh", "-c", script).Output()

	if err != nil {
		result <- AnswerNetIface{
			iface,
			map[string]int{
				"TX": 0,
				"RX": 0,
			},
			err,
		}
	} else {
		arrayValue := strings.Split(string(stdout), "\n")
		tx, _ := strconv.Atoi(arrayValue[0])
		rx, _ := strconv.Atoi(arrayValue[1])

		result <- AnswerNetIface{
			iface,
			map[string]int{
				"TX": tx,
				"RX": rx,
			},
			nil,
		}
	}
}

//CreateLoadNetworkInterface формирует информация о загрузке сетевых интерфейсов
func (sysInfo *SysInfo) CreateLoadNetworkInterface(done chan<- struct{}, errMsg chan<- error) { //done chan<- struct{}, errMsg chan<- error) {
	stdout, err := exec.Command("sh", "-c", "ifconfig -s|awk '{print $1}'|sed '1d'").Output()
	if err != nil {
		errMsg <- err
		done <- struct{}{}

		return
	}

	var listIface []string
	listInterfaceString := strings.Split(string(stdout), "\n")
	for id := range listInterfaceString {
		if listInterfaceString[id] == "lo" || listInterfaceString[id] == "" {
			continue
		}
		listIface = append(listIface, listInterfaceString[id])
	}

	var answer = make(chan AnswerNetIface, len(listIface))
	defer close(answer)

	for _, iface := range listIface {
		go getLoadingNetworkInterface(answer, iface)
	}

	var answerNetIface AnswerNetIface
	mapLoadNetwork := make(map[string]map[string]int)

	for i := 0; i < len(listIface); i++ {
		answerNetIface = <-answer

		if answerNetIface.ErrMsg != nil {
			errMsg <- answerNetIface.ErrMsg
		}

		mapLoadNetwork[answerNetIface.Iface] = answerNetIface.Data
	}

	sysInfo.Info.LoadNetwork = mapLoadNetwork

	done <- struct{}{}
}
