package configure

/*
* Описание типа в котором хранятся параметры и выполняемые задачи
* приложения
*
* Версия 0.1, дата релиза 03.04.2019
* */

//StoreMemoryTasksApplication параметры и задачи приложения
type StoreMemoryTasksApplication struct {
	settings        SettingsApplication
	filtrationTasks map[string]*FiltrationTasks
	downloadTasks   map[string]*DownloadTasks
}

//SettingsApplication параметры приложения
type SettingsApplication struct {
	MaxCountProcessFiltration int8
	EnableTelemetry           bool
}

//FiltrationTasks описание параметров задач по фильтрации
type FiltrationTasks struct{}

//DownloadTasks описание параметров задач по выгрузке файлов
type DownloadTasks struct{}

//NewRepositorySMTA создание нового репозитория
func NewRepositorySMTA() *StoreMemoryTasksApplication {
	smta := StoreMemoryTasksApplication{}
	smta.settings = SettingsApplication{}
	smta.filtrationTasks = map[string]*FiltrationTasks{}
	smta.downloadTasks = map[string]*DownloadTasks{}

	return &smta
}

//SetSettingApplication устанавливает параметры приложения
func (smta *StoreMemoryTasksApplication) SetSettingApplication(s SettingsApplication) {
	smta.settings = s
}

//GetSettingApplication получает параметры приложения
func (smta *StoreMemoryTasksApplication) GetSettingApplication() SettingsApplication {
	return smta.settings
}
