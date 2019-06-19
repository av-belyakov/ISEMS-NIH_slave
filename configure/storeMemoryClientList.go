package configure

/*
* Модуль хранения списков подключенных клиентов
*
* Версия 0.1, дата релиза 28.03.2019
* */

//SourceSetting параметры источника
type SourceSetting struct {
	ConnectionStatus  bool
	IP                string
	Port              string
	DateLastConnected int64 //Unix time
	Token             string
	AccessIsAllowed   bool              //разрешен ли доступ, по умолчанию false (при проверке токена ставится true если он верен)
	CurrentTasks      map[string]string // задачи для данного источника,
}

//WssConnection дескриптор соединения по протоколу websocket
/*type WssConnection struct {
	Link *websocket.Conn
	//mu   sync.Mutex
}*/

//sourcesListSetting настройки источников, ключ ID источника
type sourcesListSetting map[string]SourceSetting

//sourcesListConnection дескрипторы соединения с источниками по протоколу websocket
type sourcesListConnection map[string]WssConnection

//InformationSourcesList информация об источниках
type InformationSourcesList struct {
	sourcesListSetting
	sourcesListConnection
}

//NewRepositoryISL инициализация хранилища
func NewRepositoryISL() *InformationSourcesList {
	isl := InformationSourcesList{}
	isl.sourcesListSetting = sourcesListSetting{}
	isl.sourcesListConnection = sourcesListConnection{}

	return &isl
}

//AddSourceSettings добавить настройки источника
func (isl *InformationSourcesList) AddSourceSettings(id string, settings SourceSetting) {
	isl.sourcesListSetting[id] = settings
}

//SearchSourceIPAndToken поиск id источника по его ip и токену
func (isl *InformationSourcesList) SearchSourceIPAndToken(ip, token string) (string, bool) {
	for id, s := range isl.sourcesListSetting {
		if s.IP == ip && s.Token == token {
			//разрешаем соединение с данным источником
			s.AccessIsAllowed = true

			return id, true
		}
	}

	return "", false
}

//GetSourceIDOnIP получить ID источника по его IP
func (isl *InformationSourcesList) GetSourceIDOnIP(ip string) (string, bool) {
	for id, s := range isl.sourcesListSetting {
		if s.IP == ip {
			return id, true
		}
	}

	return "", false
}

//GetSourceSetting получить все настройки источника по его id
func (isl *InformationSourcesList) GetSourceSetting(id string) (*SourceSetting, bool) {
	if s, ok := isl.sourcesListSetting[id]; ok {
		return &s, true
	}

	return &SourceSetting{}, false
}

//GetSourceList возвращает список источников
func (isl *InformationSourcesList) GetSourceList() *map[string]SourceSetting {
	sl := map[string]SourceSetting{}

	for id, ss := range isl.sourcesListSetting {
		sl[id] = ss
	}

	return &sl
}

//SetAccessIsAllowed устанавливает статус позволяющий продолжать wss соединение
func (isl *InformationSourcesList) SetAccessIsAllowed(id string) {
	if s, ok := isl.sourcesListSetting[id]; ok {
		s.AccessIsAllowed = true
		isl.sourcesListSetting[id] = s
	}
}

//GetCountSources возвращает общее количество источников
func (isl InformationSourcesList) GetCountSources() int {
	return len(isl.sourcesListSetting)
}

//GetListsConnectedAndDisconnectedSources возвращает списки источников подключенных и не подключенных
func (isl InformationSourcesList) GetListsConnectedAndDisconnectedSources() (listConnected, listDisconnected map[string]string) {
	listConnected, listDisconnected = map[string]string{}, map[string]string{}

	for id, source := range isl.sourcesListSetting {
		if source.ConnectionStatus {
			listConnected[id] = source.IP
		} else {
			listDisconnected[id] = source.IP
		}
	}

	return listConnected, listDisconnected
}

//GetListSourcesWhichTaskExecuted возвращает список источников на которых выполняются задачи
func (isl InformationSourcesList) GetListSourcesWhichTaskExecuted() (let map[string]string) {
	for id, source := range isl.sourcesListSetting {
		if len(source.CurrentTasks) > 0 {
			let[id] = source.IP
		}
	}

	return let
}

//GetListTasksPerformedSourceByType получить список выполняемых на источнике задач по типу
func (isl InformationSourcesList) GetListTasksPerformedSourceByType(id string, taskType string) []string {
	taskList := []string{}
	if s, ok := isl.sourcesListSetting[id]; ok {
		for tid, info := range s.CurrentTasks {
			if info == taskType {
				taskList = append(taskList, tid)
			}
		}
	}

	return taskList
}
