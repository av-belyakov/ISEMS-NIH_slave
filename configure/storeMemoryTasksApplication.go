package configure

import (
	"time"

	"github.com/gorilla/websocket"
)

/*
* Описание типа в котором хранятся параметры и выполняемые задачи
* приложения
*
* Версия 0.1, дата релиза 03.04.2019
* */

//StoreMemoryApplication параметры и задачи приложения
type StoreMemoryApplication struct {
	applicationSettings ApplicationSettings
	clientSettings      map[string]ClientSettings
	clientTasks         map[string]TasksList
	clientLink          map[string]WssConnection
}

//ApplicationSettings параметры приложения
type ApplicationSettings struct {
	StorageFolders []string
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
type FiltrationTasks struct{}

//DownloadTasks описание параметров задач по выгрузке файлов
type DownloadTasks struct{}

//NewRepositorySMA создание нового репозитория
func NewRepositorySMA() *StoreMemoryApplication {
	sma := StoreMemoryApplication{}

	sma.applicationSettings = ApplicationSettings{}
	sma.clientSettings = map[string]ClientSettings{}
	sma.clientTasks = map[string]TasksList{}
	sma.clientLink = map[string]WssConnection{}

	return &sma
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

//AddTask добавить задачу
func (sma *StoreMemoryApplication) AddTask(taskType string) {}
