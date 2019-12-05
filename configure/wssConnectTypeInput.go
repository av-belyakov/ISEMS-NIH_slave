package configure

/*
* Типы JSON сообщений принимаемых от клиента
* */

//DetailInfoMsgPing подробная информация
// EnableTelemetry - включить телеметрию
// StorageFolders - директория для хранения файлов
// TypeAreaNetwork - тип сети
type DetailInfoMsgPing struct {
	EnableTelemetry bool     `json:"enableTelemetry"`
	StorageFolders  []string `json:"storageFolders"`
	TypeAreaNetwork string   `json:"typeAreaNetwork"`
}

//MsgTypePing сообщение типа ping
type MsgTypePing struct {
	MsgType string            `json:"messageType"`
	Info    DetailInfoMsgPing `json:"info"`
}

/* ПАРАМЕТРЫ ФИЛЬТРАЦИИ */

//TypeUnixSocketInteraction получить токен (для Unix сокетов)
type TypeUnixSocketInteraction struct {
	Token string `json:"token"`
}

//MsgTypeFiltrationControl сообщение для запуска процесса фильтрации
type MsgTypeFiltrationControl struct {
	MsgType string                    `json:"messageType"`
	Info    SettingsFiltrationControl `json:"info"`
}

//SettingsFiltrationControl описание параметров для запуска процесса фильтрации
// TaskID - идентификатор задачи
// Command - команда 'start'/'stop'
// IndexIsFound - найдены ли индексы
// CountIndexFiles - количество файлов найденных в результате поиска по индексу
// NumberMessagesFrom - количество сообщений из... например, 1 из 3
// Options - параметры фильтрации, заполняются ТОЛЬКО в сообщении где NumberMessageFrom[0,N]
type SettingsFiltrationControl struct {
	TaskID                 string                                      `json:"id"`
	Command                string                                      `json:"c"`
	Options                FiltrationControlCommonParametersFiltration `json:"o"`
	IndexIsFound           bool                                        `json:"iif"`
	CountIndexFiles        int                                         `json:"cif"`
	NumberMessagesFrom     [2]int                                      `json:"nmf"`
	ListFilesReceivedIndex map[string][]string                         `json:"lfri"`
}

//FiltrationControlCommonParametersFiltration описание параметров фильтрации
type FiltrationControlCommonParametersFiltration struct {
	ID       int                                       `json:"id"`
	DateTime DateTimeParameters                        `json:"dt"`
	Protocol string                                    `json:"p"`
	Filters  FiltrationControlParametersNetworkFilters `json:"f"`
}

//DateTimeParameters параметры времени
type DateTimeParameters struct {
	Start int64 `json:"s"`
	End   int64 `json:"e"`
}

//FiltrationControlParametersNetworkFilters параметры сетевых фильтров
type FiltrationControlParametersNetworkFilters struct {
	IP      FiltrationControlIPorNetorPortParameters `json:"ip"`
	Port    FiltrationControlIPorNetorPortParameters `json:"pt"`
	Network FiltrationControlIPorNetorPortParameters `json:"nw"`
}

//FiltrationControlIPorNetorPortParameters параметры для ip или network
type FiltrationControlIPorNetorPortParameters struct {
	Any []string `json:"any"`
	Src []string `json:"src"`
	Dst []string `json:"dst"`
}

/* ПАРАМЕТРЫ СКАЧИВАНИЯ ФАЙЛОВ */

//MsgTypeDownloadControl сообщение типа 'download files'
type MsgTypeDownloadControl struct {
	MsgType string                `json:"messageType"`
	Info    DetailInfoMsgDownload `json:"info"`
}

//DetailInfoMsgDownload подробная информация
// TaskID - ID задачи
// Command - статус выполняемой задачи
//  - 'give me the file' (master -> slave), запрос файла
//  - 'ready to receive file' (master -> salve), подтверждение готовности приема файла
//  - 'ready for the transfer' (slave -> master), подтверждение готовности передачи
//  - 'file transfer not possible' (slave -> master), сообщение о невозможности передачи
//  - 'file transfer complete' (slave -> master), сообщение о завершении передачи
// PathDirStorage - директория в которой хранятся файлы на источнике
// FileOptions - параметры файла
type DetailInfoMsgDownload struct {
	TaskID         string              `json:"tid"`
	Command        string              `json:"c"`
	PathDirStorage string              `json:"pds"`
	FileOptions    DownloadFileOptions `json:"fo"`
}

//DownloadFileOptions параметры загружаемого файла
// Name - название файла
// Size - размер файла
// Hex - контрольная сумма файла
// NumChunk - кол-во передаваемых кусочков
// ChunkSize - размер передаваемого кусочка
type DownloadFileOptions struct {
	Name      string `json:"n"`
	Size      int64  `json:"sz"`
	Hex       string `json:"hex"`
	NumChunk  int    `json:"nc"`
	ChunkSize int    `json:"cs"`
}
