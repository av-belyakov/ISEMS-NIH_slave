package configure

/*
* Типы JSON сообщений отправляемых клиенту
* */

//MsgTypeFiltration сообщение типа ping
type MsgTypeFiltration struct {
	MsgType string                  `json:"messageType"`
	Info    DetailInfoMsgFiltration `json:"info"`
}

//DetailInfoMsgFiltration подробная информация
// TaskID - ID задачи
// TaskStatus - статус выполняемой задачи
// NumberFilesToBeFiltered - количество файлов подходящих под параметры фильтрации
// SizeFilesToBeFiltered — общий размер файлов подходящих под параметры фильтрации
// CountDirectoryFiltartion — количество директорий по которым выполняется фильт.
// NumberFilesProcessed — количество обработанных файлов
// NumberFilesFound — количество найденных файлов
// SizeFilesFound — общий размер найденных файлов
// PathStorageSource — путь до директории в которой сохраняются файлы при
// FoundFileName - имя файла, найденного в результате фильтрации сет. трафика
// FoundSizeFile - размер файла, найденного в результате фильтрации сет. трафика
// FoundFilesInformation - информация о файлах, ключ - имя файла
type DetailInfoMsgFiltration struct {
	TaskID                   string                            `json:"tid"`
	TaskStatus               string                            `json:"ts"`
	NumberFilesToBeFiltered  int                               `json:"nfbef"`
	SizeFilesToBeFiltered    uint64                            `json:"sfbef"`
	CountDirectoryFiltartion int                               `json:"cdf"`
	NumberFilesProcessed     int                               `json:"nfp"`
	NumberFilesFound         int                               `json:"nff"`
	SizeFilesFound           uint64                            `json:"sff"`
	PathStorageSource        string                            `json:"pss"`
	FoundFilesInformation    map[string]*FoundFilesInformation `json:"ffi"`
}

//FoundFilesInformation подробная информация о файлах
// Size - размер файла
// Hex - хеш сумма файла
type FoundFilesInformation struct {
	Size uint64 `json:"s"`
	Hex  string `json:"h"`
}

//MsgTypeError сообщение отправляемое при возникновении ошибки
type MsgTypeError struct {
	MsgType string             `json:"messageType"`
	Info    DetailInfoMsgError `json:"info"`
}

//DetailInfoMsgError детальное описание ошибки
// TaskID - id задачи
// ErrorName - наименование ошибки
// ErrorDescription - детальное описание ошибки
type DetailInfoMsgError struct {
	TaskID           string `json:"tid"`
	ErrorName        string `json:"en"`
	ErrorDescription string `json:"ed"`
}

//MsgTypeNotification информационное сообщение
type MsgTypeNotification struct {
	MsgType string                    `json:"messageType"`
	Info    DetailInfoMsgNotification `json:"info"`
}

//DetailInfoMsgNotification информационное сообщение, подробная информация
// TaskID - id задачи
// Section - раздел обработки заданий
// TypeActionPerformed - тип выполняемого действия при обработке задания
// CriticalityMessage - тип сообщения ('info'/'success'/'warning'/'danger')
// Description - описание сообщения
type DetailInfoMsgNotification struct {
	TaskID              string `json:"tid"`
	Section             string `json:"s"`
	TypeActionPerformed string `json:"tap"`
	CriticalityMessage  string `json:"cm"`
	Description         string `json:"d"`
}
