package configure

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

/*
* Описание типа в котором хранятся параметры и выполняемые задачи приложения
*
* Версия 0.32, дата релиза 22.10.2019
* */

//StoreMemoryApplication параметры и задачи приложения
// map[string] = clientID
type StoreMemoryApplication struct {
	applicationSettings ApplicationSettings
	clientSettings      map[string]ClientSettings
	clientTasks         map[string]TasksList
	clientLink          map[string]WssConnection
	chanReqSettingsTask chan chanReqSettingsTask
}

//ApplicationSettings параметры приложения
type ApplicationSettings struct {
	TypeAreaNetwork string
	StorageFolders  []string
}

//ClientSettings настройки индивидуальные для клиента
type ClientSettings struct {
	ConnectionStatus  bool
	IP                string
	Port              string
	Token             string
	DateLastConnected int64 //Unix time
	AccessIsAllowed   bool
	SendsTelemetry    bool
}

//TasksList список задач
// ключом 'filtrationTasks' и 'downloadTasks' является идентификатор задачи
type TasksList struct {
	filtrationTasks map[string]*FiltrationTasks
	downloadTasks   map[string]*DownloadTasks
}

//WssConnection дескриптор соединения по протоколу websocket
type WssConnection struct {
	Link *websocket.Conn
	//mu   sync.Mutex
}

//FiltrationTasks описание параметров задач по фильтрации
// DateTimeStart, DateTimeEnd - временной интервал для фильтрации
// Protocol - тип протокола транспортного уровня (TCP, UDP)
// Filters - параметры фильтрации
// Status - состояние задачи
// UseIndex - используется ли индекс
// NumberFilesMeetFilterParameters - количество найденных по индексам файлов (или просто найденных файлов, если не используется индекс)
// NumberProcessedFiles - количество обработанных файлов
// NumberFilesFoundResultFiltering - количество найденных в результате фильтрации файлов
// NumberErrorProcessedFiles - количество файлов не обработанных при обработке которых возникли ошибки
// SizeFilesMeetFilterParameters - общий размер всех найденных по индексам файлов
// SizeFilesFoundResultFiltering - общий размер всех найденных файлов
// FileStorageDirectory - директория для хранения файлов
// ListChanStopFiltration - список каналов для останова задачи
// ListFiles - список файлов найденных в результате поиска по индексам
type FiltrationTasks struct {
	DateTimeStart, DateTimeEnd      int64
	Protocol                        string
	Filters                         FiltrationControlParametersNetworkFilters
	Status                          string
	UseIndex                        bool
	NumberFilesMeetFilterParameters int
	NumberFilesFoundResultFiltering int
	NumberProcessedFiles            int
	NumberErrorProcessedFiles       int
	SizeFilesMeetFilterParameters   int64
	SizeFilesFoundResultFiltering   int64
	FileStorageDirectory            string
	ListChanStopFiltration          []chan struct{}
	ListFiles                       map[string][]string
}

//DownloadTasks описание параметров задач по выгрузке файлов
// FileName - имя выгружаемого файла
// FileSize - размер выгружаемого файла
// FileHex - хеш-сумма выгружаемого файла
// NumFileChunk - количество частей файла
// NumChunkSent - количество оправленых частей
// SizeFileChunk - размер части файла
// StrHex - хеш-строка идентифицирующая часть файла
// DirectiryPathStorage - путь к директории для хранения файлов
// ChanStopReadFile - канал для останова чтения и передачи файла
type DownloadTasks struct {
	FileName             string
	FileSize             int64
	FileHex              string
	NumFileChunk         int
	NumChunkSent         int
	SizeFileChunk        int
	StrHex               string
	DirectiryPathStorage string
	ChanStopReadFile     chan struct{}
}

type chanReqSettingsTask struct {
	ClientID, TaskID, TaskType, ActionType string
	ChanRespons                            chan chanResSettingsTask
	Parameters                             interface{}
}

type chanResSettingsTask struct {
	Error      error
	Parameters interface{}
}

//NewRepositorySMA создание нового репозитория
func NewRepositorySMA() *StoreMemoryApplication {
	sma := StoreMemoryApplication{}

	sma.applicationSettings = ApplicationSettings{}
	sma.clientSettings = map[string]ClientSettings{}
	sma.clientTasks = map[string]TasksList{}
	sma.clientLink = map[string]WssConnection{}

	sma.chanReqSettingsTask = make(chan chanReqSettingsTask)

	go func() {
		for msg := range sma.chanReqSettingsTask {
			switch msg.TaskType {
			case "check exist client id":
				if _, ok := sma.clientTasks[msg.ClientID]; !ok {
					msg.ChanRespons <- chanResSettingsTask{
						Error: fmt.Errorf("client with ID %v not found", msg.ClientID),
					}

					close(msg.ChanRespons)

					continue
				}

				msg.ChanRespons <- chanResSettingsTask{}

				close(msg.ChanRespons)

			/*case "check filtration task exist":
				if _, ok := sma.clientTasks[msg.ClientID]; !ok {
					msg.ChanRespons <- chanResSettingsTask{
						Error: fmt.Errorf("tasks filtration for client with ID %v not found", msg.ClientID),
					}

					close(msg.ChanRespons)

					continue
				}

				if _, ok := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID]; !ok {
					msg.ChanRespons <- chanResSettingsTask{
						Error: fmt.Errorf("tasks filtration with ID %v not found", msg.TaskID),
					}

					close(msg.ChanRespons)

					continue
				}

				msg.ChanRespons <- chanResSettingsTask{}

				close(msg.ChanRespons)

			case "check download task exist":
				if _, ok := sma.clientTasks[msg.ClientID]; !ok {
					msg.ChanRespons <- chanResSettingsTask{
						Error: fmt.Errorf("tasks download for client with ID %v not found", msg.ClientID),
					}

					close(msg.ChanRespons)

					continue
				}

				if _, ok := sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID]; !ok {
					msg.ChanRespons <- chanResSettingsTask{
						Error: fmt.Errorf("tasks download with ID %v not found", msg.TaskID),
					}

					close(msg.ChanRespons)

					continue
				}

				msg.ChanRespons <- chanResSettingsTask{}

				close(msg.ChanRespons)
			*/
			case "filtration":
				switch msg.ActionType {
				case "get task information":
					if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID],
					}

					close(msg.ChanRespons)

				case "inc num proc files":
					if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberProcessedFiles + 1
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberProcessedFiles = num

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}

					close(msg.ChanRespons)

				case "inc num not proc files":
					if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberErrorProcessedFiles + 1
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberErrorProcessedFiles = num

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}

					close(msg.ChanRespons)

				case "inc num found files":
					if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberFilesFoundResultFiltering + 1
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].NumberFilesFoundResultFiltering = num

					if fileSize, ok := msg.Parameters.(int64); ok {
						size := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].SizeFilesFoundResultFiltering + fileSize
						sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].SizeFilesFoundResultFiltering = size
					}

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}

					close(msg.ChanRespons)

				case "add new chan to stop filtration":
					if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					if csf, ok := msg.Parameters.(chan struct{}); ok {
						lcsf := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].ListChanStopFiltration

						lcsf = append(lcsf, csf)
						sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].ListChanStopFiltration = lcsf
					}

					msg.ChanRespons <- chanResSettingsTask{}

					close(msg.ChanRespons)

				case "delete task":
					resMsg := chanResSettingsTask{}
					if err := sma.checkForFilteringTask(msg.ClientID, msg.TaskID); err != nil {
						resMsg.Error = err
					}

					delete(sma.clientTasks[msg.ClientID].filtrationTasks, msg.TaskID)

					msg.ChanRespons <- resMsg

					close(msg.ChanRespons)
				}

			case "download":
				switch msg.ActionType {
				case "add chan stop read file":
					if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					if csrf, ok := msg.Parameters.(chan struct{}); ok {
						sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID].ChanStopReadFile = csrf
					}

					msg.ChanRespons <- chanResSettingsTask{}

					close(msg.ChanRespons)

				case "get task information":
					if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID],
					}

					close(msg.ChanRespons)

				case "inc num chunk sent":
					if err := sma.checkForDownloadingTask(msg.ClientID, msg.TaskID); err != nil {
						msg.ChanRespons <- chanResSettingsTask{
							Error: err,
						}

						close(msg.ChanRespons)

						continue
					}

					num := sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID].NumChunkSent + 1
					sma.clientTasks[msg.ClientID].downloadTasks[msg.TaskID].NumChunkSent = num

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}

					close(msg.ChanRespons)

				case "delete task":
					delete(sma.clientTasks[msg.ClientID].downloadTasks, msg.TaskID)

				case "delete all tasks":
					for taskID := range sma.clientTasks[msg.ClientID].downloadTasks {
						delete(sma.clientTasks[msg.ClientID].downloadTasks, taskID)
					}

				}
			}
		}
	}()

	return &sma
}

//checkExistClientID проверяет существование информации о клиенте
func (sma *StoreMemoryApplication) checkExistClientID(clientID string) error {
	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskType:    "check exist client id",
		ChanRespons: chanRes,
	}

	return (<-chanRes).Error
}

//checkForFilteringTask проверяет наличие задачи
func (sma *StoreMemoryApplication) checkForFilteringTask(clientID, taskID string) error {
	if _, ok := sma.clientTasks[clientID]; !ok {
		return fmt.Errorf("tasks filtration for client with ID %v not found", clientID)
	}

	if _, ok := sma.clientTasks[clientID].filtrationTasks[taskID]; !ok {
		return fmt.Errorf("tasks filtration with ID %v not found", taskID)
	}

	return nil
}

//checkForDownloadingTask проверяет наличие задачи
func (sma *StoreMemoryApplication) checkForDownloadingTask(clientID, taskID string) error {
	if _, ok := sma.clientTasks[clientID]; !ok {
		return fmt.Errorf("tasks download for client with ID %v not found", clientID)
	}

	if _, ok := sma.clientTasks[clientID].downloadTasks[taskID]; !ok {
		return fmt.Errorf("tasks download with ID %v not found", taskID)
	}

	return nil
}

//checkTaskExist проверяет существование задачи
/*func (sma *StoreMemoryApplication) checkTaskExist(clientID, taskID, taskType string) error {
	chanRes := make(chan chanResSettingsTask)

	tt := "check filtration task exist"

	if taskType == "download" {
		tt = "check download task exist"
	}

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    tt,
		ChanRespons: chanRes,
	}

	return (<-chanRes).Error
}*/

/* параметры приложения */
/*----------------------*/

//SetApplicationSetting устанавливает параметры приложения
func (sma *StoreMemoryApplication) SetApplicationSetting(as ApplicationSettings) {
	sma.applicationSettings = as
}

//GetApplicationSetting возвращает параметры приложения
func (sma StoreMemoryApplication) GetApplicationSetting() ApplicationSettings {
	return sma.applicationSettings
}

/* параметры клиента */
/*--------------------*/

//SetClientSetting устанавливает параметры клиента
func (sma *StoreMemoryApplication) SetClientSetting(clientID string, settings ClientSettings) {
	sma.clientSettings[clientID] = settings

	if _, ok := sma.clientTasks[clientID]; !ok {
		sma.clientTasks[clientID] = TasksList{
			filtrationTasks: map[string]*FiltrationTasks{},
			downloadTasks:   map[string]*DownloadTasks{},
		}
	}
}

//GetClientSetting передает параметры клиента
func (sma StoreMemoryApplication) GetClientSetting(clientID string) (ClientSettings, bool) {
	cs, ok := sma.clientSettings[clientID]
	return cs, ok
}

//GetAllClientSettings передает параметры по всем клиентам
func (sma StoreMemoryApplication) GetAllClientSettings() map[string]ClientSettings {
	return sma.clientSettings
}

//GetClientIDOnIP получить ID источника по его IP
func (sma StoreMemoryApplication) GetClientIDOnIP(clientIP string) (string, bool) {
	for id, s := range sma.clientSettings {
		if s.IP == clientIP {
			return id, true
		}
	}

	return "", false
}

//GetAccessIsAllowed передает значение подтверждающее или отклоняющее права доступа источника
func (sma StoreMemoryApplication) GetAccessIsAllowed(clientIP string) bool {
	for _, s := range sma.clientSettings {
		if s.IP == clientIP {
			return s.AccessIsAllowed
		}
	}

	return false
}

//DeleteClientSetting удаляет параметры клиента
func (sma *StoreMemoryApplication) DeleteClientSetting(clientID string) {
	delete(sma.clientSettings, clientID)
}

//ChangeSourceConnectionStatus изменить состояние клиента
func (sma *StoreMemoryApplication) ChangeSourceConnectionStatus(clientID string, status bool) bool {
	if s, ok := sma.clientSettings[clientID]; ok {
		s.ConnectionStatus = status

		if status {
			s.DateLastConnected = time.Now().Unix()
		} else {
			s.AccessIsAllowed = false
		}
		sma.clientSettings[clientID] = s

		return true
	}

	return false
}

/* параметры сетевого соединения */
/*-------------------------------*/

//SendWsMessage используется для отправки сообщений через протокол websocket (применяется Mutex)
func (wssc *WssConnection) SendWsMessage(t int, v []byte) error {
	/*wssc.mu.Lock()
	defer wssc.mu.Unlock()*/

	return wssc.Link.WriteMessage(t, v)
}

//GetClientsListConnection получить список всех соединений
func (sma *StoreMemoryApplication) GetClientsListConnection() map[string]WssConnection {
	return sma.clientLink
}

//AddLinkWebsocketConnect добавить линк соединения по websocket
func (sma *StoreMemoryApplication) AddLinkWebsocketConnect(clientIP string, lwsc *websocket.Conn) {
	sma.clientLink[clientIP] = WssConnection{
		Link: lwsc,
	}
}

//DelLinkWebsocketConnection удаление дескриптора соединения при отключении источника
func (sma *StoreMemoryApplication) DelLinkWebsocketConnection(clientIP string) {
	delete(sma.clientLink, clientIP)
}

//GetLinkWebsocketConnect получить линк соединения по websocket
func (sma *StoreMemoryApplication) GetLinkWebsocketConnect(clientIP string) (*WssConnection, bool) {
	conn, ok := sma.clientLink[clientIP]

	return &conn, ok
}

/* параметры выполняемых задач */
/*-----------------------------*/

//AddTaskFiltration добавить задачу
func (sma *StoreMemoryApplication) AddTaskFiltration(clientID, taskID string, ft *FiltrationTasks) {
	sma.clientTasks[clientID].filtrationTasks[taskID] = ft
}

//GetListTasksFiltration получить список задач по фильтрации выполняемых данным пользователем
func (sma *StoreMemoryApplication) GetListTasksFiltration(clientID string) (map[string]*FiltrationTasks, bool) {
	tasks, ok := sma.clientTasks[clientID]
	if !ok {
		return nil, false
	}

	return tasks.filtrationTasks, true
}

//GetInfoTaskFiltration получить всю информацию о задаче выполняемой пользователем
func (sma *StoreMemoryApplication) GetInfoTaskFiltration(clientID, taskID string) (*FiltrationTasks, error) {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {

		fmt.Printf("SMTA func 'GetInfoTaskFiltration', ERROR: %v\n", err)

		return nil, err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "filtration",
		ActionType:  "get task information",
		ChanRespons: chanRes,
	}

	res := <-chanRes
	if info, ok := res.Parameters.(*FiltrationTasks); ok {
		return info, res.Error
	}

	return nil, res.Error
}

//SetInfoTaskFiltration устанавливает новое значение некоторых параметров
func (sma *StoreMemoryApplication) SetInfoTaskFiltration(clientID, taskID string, settings map[string]interface{}) error {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {

		fmt.Printf("SMTA func 'SetInfoTaskFiltration', ERROR: %v\n", err)

		return err
	}*/

	for k, v := range settings {
		switch k {
		case "FileStorageDirectory":
			if fsd, ok := v.(string); ok {
				sma.clientTasks[clientID].filtrationTasks[taskID].FileStorageDirectory = fsd
			}

		case "NumberFilesMeetFilterParameters", "NumberProcessedFiles", "NumberErrorProcessedFiles":
			if count, ok := v.(int); ok {
				if k == "NumberFilesMeetFilterParameters" {
					sma.clientTasks[clientID].filtrationTasks[taskID].NumberFilesMeetFilterParameters = count
				}
				if k == "NumberProcessedFiles" {
					sma.clientTasks[clientID].filtrationTasks[taskID].NumberProcessedFiles = count
				}
				if k == "NumberErrorProcessedFiles" {
					sma.clientTasks[clientID].filtrationTasks[taskID].NumberErrorProcessedFiles = count
				}
			}

		case "SizeFilesMeetFilterParameters":
			if size, ok := v.(int64); ok {
				sma.clientTasks[clientID].filtrationTasks[taskID].SizeFilesMeetFilterParameters = size
			}

		case "Status":
			if status, ok := v.(string); ok {
				sma.clientTasks[clientID].filtrationTasks[taskID].Status = status
			}

		default:
			return errors.New("you cannot change the value, undefined passed parameter")
		}
	}

	return nil
}

//IncrementNumProcessedFiles увеличивает кол-во обработанных файлов
func (sma *StoreMemoryApplication) IncrementNumProcessedFiles(clientID, taskID string) (int, error) {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {

		fmt.Printf("SMTA func 'IncrementNumProcessedFiles', ERROR: %v\n", err)

		return 0, err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "filtration",
		ActionType:  "inc num proc files",
		ChanRespons: chanRes,
	}

	res := <-chanRes
	if num, ok := res.Parameters.(int); ok {
		return num, res.Error
	}

	return 0, res.Error
}

//IncrementNumNotFoundIndexFiles увеличивает кол-во не обработанных файлов
func (sma *StoreMemoryApplication) IncrementNumNotFoundIndexFiles(clientID, taskID string) (int, error) {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {

		fmt.Printf("SMTA func 'IncrementNumNotFoundIndexFiles', ERROR: %v\n", err)

		return 0, err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "filtration",
		ActionType:  "inc num found files",
		ChanRespons: chanRes,
	}

	res := <-chanRes
	if num, ok := res.Parameters.(int); ok {
		return num, res.Error
	}

	return 0, res.Error
}

//IncrementNumFoundFiles увеличивает кол-во найденных, в результате фильтрации, файлов и их общий размер
func (sma *StoreMemoryApplication) IncrementNumFoundFiles(clientID, taskID string, fileSize int64) (int, error) {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {

		fmt.Printf("SMTA func 'IncrementNumFoundFiles', ERROR: %v\n", err)

		return 0, err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "filtration",
		ActionType:  "inc num found files",
		ChanRespons: chanRes,
		Parameters:  fileSize,
	}

	res := <-chanRes
	if num, ok := res.Parameters.(int); ok {
		return num, res.Error
	}

	return 0, res.Error
}

//AddNewChanStopProcessionFiltrationTask добавляет новый канал для останова процессов задачи фильтрации файлов
func (sma *StoreMemoryApplication) AddNewChanStopProcessionFiltrationTask(clientID, taskID string, ncsf chan struct{}) error {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {
		return err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "filtration",
		ActionType:  "add new chan to stop filtration",
		ChanRespons: chanRes,
		Parameters:  ncsf,
	}

	return (<-chanRes).Error
}

//AddFileToListFilesFiltrationTask добавить в основной список часть списка найденных, в том числе и по индексам, файлов
func (sma *StoreMemoryApplication) AddFileToListFilesFiltrationTask(clientID, taskID string, fl map[string][]string) (int, error) {
	var numberFilesMeetFilterParameters int

	if _, ok := sma.clientTasks[clientID]; !ok {
		return numberFilesMeetFilterParameters, fmt.Errorf("tasks for client with ID %v not found", clientID)
	}

	if _, ok := sma.clientTasks[clientID].filtrationTasks[taskID]; !ok {
		return numberFilesMeetFilterParameters, fmt.Errorf("tasks with ID %v not found", taskID)
	}

	list := sma.clientTasks[clientID].filtrationTasks[taskID].ListFiles
	if len(list) == 0 {
		list = make(map[string][]string, len(list))
	}

	for k, v := range fl {
		if _, ok := list[k]; !ok {
			list[k] = make([]string, 0, 100)
		}

		list[k] = append(list[k], v...)

		numberFilesMeetFilterParameters += len(v)
	}

	sma.clientTasks[clientID].filtrationTasks[taskID].ListFiles = list

	return numberFilesMeetFilterParameters, nil
}

//DelTaskFiltration удаление выбранной задачи
func (sma *StoreMemoryApplication) DelTaskFiltration(clientID, taskID string) error {
	/*if err := sma.checkTaskExist(clientID, taskID, "filtration"); err != nil {

		fmt.Printf("SMTA func 'DelTaskFiltration', ERROR: %v\n", err)

		return err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "filtration",
		ActionType:  "delete task",
		ChanRespons: chanRes,
	}

	return (<-chanRes).Error
}

//AddTaskDownload добавить задачу
func (sma *StoreMemoryApplication) AddTaskDownload(clientID, taskID string, dt *DownloadTasks) {
	sma.clientTasks[clientID].downloadTasks[taskID] = dt
}

//AddChanStopReadFileTaskDownload добавляет канал для останова чтения и передачи файла
func (sma *StoreMemoryApplication) AddChanStopReadFileTaskDownload(clientID, taskID string, csrf chan struct{}) error {
	/*if err := sma.checkTaskExist(clientID, taskID, "download"); err != nil {

		fmt.Printf("SMTA func 'AddChanStopReadFileTaskDownload', ERROR: %v\n", err)

		return err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "download",
		ActionType:  "add chan stop read file",
		Parameters:  csrf,
		ChanRespons: chanRes,
	}

	return (<-chanRes).Error
}

//GetInfoTaskDownload получить всю информацию о задаче выполняемой пользователем
func (sma *StoreMemoryApplication) GetInfoTaskDownload(clientID, taskID string) (*DownloadTasks, error) {
	/*if err := sma.checkTaskExist(clientID, taskID, "download"); err != nil {

		fmt.Printf("SMTA func 'GetInfoTaskDownload', ERROR: %v\n", err)

		return nil, err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "download",
		ActionType:  "get task information",
		ChanRespons: chanRes,
	}

	res := <-chanRes
	if info, ok := res.Parameters.(*DownloadTasks); ok {
		return info, res.Error
	}

	return nil, res.Error
}

//IncrementNumChunkSent увеличивает на еденицу кол-во переданных частей файла
func (sma *StoreMemoryApplication) IncrementNumChunkSent(clientID, taskID string) (int, error) {
	/*if err := sma.checkTaskExist(clientID, taskID, "download"); err != nil {

		fmt.Printf("SMTA func 'IncrementNumChunkSent', ERROR: %v\n", err)

		return 0, err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "download",
		ActionType:  "inc num chunk sent",
		ChanRespons: chanRes,
	}

	res := <-chanRes
	if num, ok := res.Parameters.(int); ok {
		return num, res.Error
	}

	return 0, res.Error
}

//DelTaskDownload удаление выбранной задачи
func (sma *StoreMemoryApplication) DelTaskDownload(clientID, taskID string) error {
	/*if err := sma.checkTaskExist(clientID, taskID, "download"); err != nil {

		fmt.Printf("SMTA func 'DelTaskDownload', ERROR: %v\n", err)

		return err
	}*/

	chanRes := make(chan chanResSettingsTask)

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:    clientID,
		TaskID:      taskID,
		TaskType:    "download",
		ActionType:  "delete task",
		ChanRespons: chanRes,
	}

	return nil
}

//DelAllTaskDownload удаляет все задачи по скачиванию файлов выполняемые для заданного пользователя
func (sma *StoreMemoryApplication) DelAllTaskDownload(clientID string) error {
	if err := sma.checkExistClientID(clientID); err != nil {
		return err
	}

	sma.chanReqSettingsTask <- chanReqSettingsTask{
		ClientID:   clientID,
		TaskType:   "download",
		ActionType: "delete all tasks",
	}

	return nil
}
