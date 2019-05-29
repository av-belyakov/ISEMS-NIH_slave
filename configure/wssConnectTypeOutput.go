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
// NumberFilesMeetFilterParameters - кол-во файлов удовлетворяющих параметрам фильтрации
// NumberProcessedFiles - кол-во обработанных файлов
// NumberFilesFoundResultFiltering - кол-во найденных, в результате фильтрации, файлов
// NumberDirectoryFiltartion - кол-во директорий по которым выполняется фильтрация
// NumberErrorProcessedFiles - кол-во не обработанных файлов или файлов обработанных с ошибками
// SizeFilesMeetFilterParameters - общий размер файлов (в байтах) удовлетворяющих параметрам фильтрации
// SizeFilesFoundResultFiltering - общий размер найденных, в результате фильтрации, файлов (в байтах)
// PathStorageSource — путь до директории в которой сохраняются файлы при
// NumberMessagesParts - порядковый номер и общее кол-во сообщений (используется ТОЛЬКО для сообщений со статусом задачи 'stop' или 'complete')
// FoundFilesInformation - информация о файлах, ключ - имя файла
type DetailInfoMsgFiltration struct {
	TaskID                          string                            `json:"tid"`
	TaskStatus                      string                            `json:"ts"`
	NumberFilesMeetFilterParameters int                               `json:"nfmfp"`
	NumberProcessedFiles            int                               `json:"npf"`
	NumberFilesFoundResultFiltering int                               `json:"nffrf"`
	NumberDirectoryFiltartion       int                               `json:"ndf"`
	NumberErrorProcessedFiles       int                               `json:"nepf"`
	SizeFilesMeetFilterParameters   int64                             `json:"sfmfp"`
	SizeFilesFoundResultFiltering   int64                             `json:"sffrf"`
	PathStorageSource               string                            `json:"pss"`
	NumberMessagesParts             [2]int                            `json:"nmp"`
	FoundFilesInformation           map[string]*InputFilesInformation `json:"ffi"`
}

//InputFilesInformation подробная информация о файлах
// Size - размер файла
// Hex - хеш сумма файла
type InputFilesInformation struct {
	Size int64  `json:"s"`
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
