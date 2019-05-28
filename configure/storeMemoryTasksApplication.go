package configure

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

/*
* Описание типа в котором хранятся параметры и выполняемые задачи
* приложения
*
* Версия 0.3, дата релиза 27.05.2019
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
	ConnectionStatus          bool
	IP                        string
	Port                      string
	Token                     string
	DateLastConnected         int64 //Unix time
	AccessIsAllowed           bool
	SendsTelemetry            bool
	MaxCountProcessFiltration int8
}

//TasksList список задач
// map[string] = taskID
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
// UseIndex - используется ли индекс
// CountIndexFiles - количество найденных по индексам файлов (или просто найденных файлов, если не используется индекс)
// SizeIndexFiles - общий размер всех найденных по индексам файлов
// CountProcessedFiles - количество обработанных файлов
// CountFoundFiles - количество найденных в результате фильтрации файлов
// CommonSizeFoundFiles - общий размер всех найденных файлов
// CountNotFoundIndexFiles - количество файлов, из перечня ListFiles, которые не были найдены по указанным путям
// FileStorageDirectory - директория для хранения файлов
// ChanStopFiltration - канал информирующий об остановке фильтрации
// ListFiles - список файлов найденных в результате поиска по индексам
type FiltrationTasks struct {
	DateTimeStart, DateTimeEnd int64
	Protocol                   string
	Filters                    FiltrationControlParametersNetworkFilters
	UseIndex                   bool
	CountIndexFiles            int
	SizeIndexFiles             int64
	CountFoundFiles            int
	CommonSizeFoundFiles       int64
	CountProcessedFiles        int
	CountNotFoundIndexFiles    int
	FileStorageDirectory       string
	ChanStopFiltration         chan struct{}
	ListFiles                  map[string][]string
}

//DownloadTasks описание параметров задач по выгрузке файлов
type DownloadTasks struct{}

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

	/*
	   IncrementNumProcessedFiles
	   IncrementNumNotFoundIndexFiles
	   IncrementNumFoundFiles
	*/

	go func() {
		for msg := range sma.chanReqSettingsTask {
			switch msg.TaskType {
			case "filtration":
				switch msg.ActionType {
				case "get task information":
					taskInfo := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID]

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: taskInfo,
					}

				case "inc num proc files":
					num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CountProcessedFiles + 1
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CountProcessedFiles = num

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}

				case "inc num not proc files":

					num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CountNotFoundIndexFiles + 1
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CountNotFoundIndexFiles = num

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}

				case "inc num found files":
					num := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CountFoundFiles + 1
					sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CountFoundFiles = num

					if fileSize, ok := msg.Parameters.(int64); ok {
						size := sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CommonSizeFoundFiles + fileSize
						sma.clientTasks[msg.ClientID].filtrationTasks[msg.TaskID].CommonSizeFoundFiles = size
					}

					msg.ChanRespons <- chanResSettingsTask{
						Parameters: num,
					}
				}

			case "download":

			}
		}
	}()

	return &sma
}

//checkTaskExist проверяет существование задачи
func (sma *StoreMemoryApplication) checkTaskExist(clientID, taskID string) error {
	if _, ok := sma.clientTasks[clientID]; !ok {
		return fmt.Errorf("tasks for client with ID %v not found", clientID)
	}

	if _, ok := sma.clientTasks[clientID].filtrationTasks[taskID]; !ok {
		return fmt.Errorf("tasks with ID %v not found", taskID)
	}

	return nil
}

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
	if err := sma.checkTaskExist(clientID, taskID); err != nil {
		return nil, err
	}

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
	if err := sma.checkTaskExist(clientID, taskID); err != nil {
		return err
	}

	for k, v := range settings {
		switch k {
		case "FileStorageDirectory":
			if fsd, ok := v.(string); ok {
				sma.clientTasks[clientID].filtrationTasks[taskID].FileStorageDirectory = fsd
			}

		case "CountIndexFiles", "CountProcessedFiles", "CountNotFoundIndexFiles":
			if count, ok := v.(int); ok {
				if k == "CountIndexFiles" {
					sma.clientTasks[clientID].filtrationTasks[taskID].CountIndexFiles = count
				}
				if k == "CountProcessedFiles" {
					sma.clientTasks[clientID].filtrationTasks[taskID].CountProcessedFiles = count
				}
				if k == "CountNotFoundIndexFiles" {
					sma.clientTasks[clientID].filtrationTasks[taskID].CountNotFoundIndexFiles = count
				}
			}

		case "SizeIndexFiles":
			if size, ok := v.(int64); ok {
				sma.clientTasks[clientID].filtrationTasks[taskID].SizeIndexFiles = size
			}

		default:
			return errors.New("you cannot change the value, undefined passed parameter")
		}
	}

	return nil
}

//IncrementNumProcessedFiles увеличивает кол-во обработанных файлов
func (sma *StoreMemoryApplication) IncrementNumProcessedFiles(clientID, taskID string) (int, error) {
	if err := sma.checkTaskExist(clientID, taskID); err != nil {
		return 0, err
	}

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
	if err := sma.checkTaskExist(clientID, taskID); err != nil {
		return 0, err
	}

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
	if err := sma.checkTaskExist(clientID, taskID); err != nil {
		return 0, err
	}

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

//AddFileToListFilesFiltrationTask добавить в основной список часть списка найденных, в том числе и по индексам, файлов
func (sma *StoreMemoryApplication) AddFileToListFilesFiltrationTask(clientID, taskID string, fl map[string][]string) (int, error) {
	var countIndexFiles int

	if _, ok := sma.clientTasks[clientID]; !ok {
		return countIndexFiles, fmt.Errorf("tasks for client with ID %v not found", clientID)
	}

	if _, ok := sma.clientTasks[clientID].filtrationTasks[taskID]; !ok {
		return countIndexFiles, fmt.Errorf("tasks with ID %v not found", taskID)
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

		countIndexFiles += len(v)
	}

	sma.clientTasks[clientID].filtrationTasks[taskID].ListFiles = list

	return countIndexFiles, nil
}

//DelTaskFiltration удаление выбранной задачи
func (sma *StoreMemoryApplication) DelTaskFiltration(clientID, taskID string) {
	if err := sma.checkTaskExist(clientID, taskID); err != nil {
		return
	}

	delete(sma.clientTasks[clientID].filtrationTasks, taskID)
}

//AddTaskDownload добавить задачу
func (sma *StoreMemoryApplication) AddTaskDownload(clientID, taskID string) {

}
