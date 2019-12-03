package configure

/*
* Описание типа конфигурационных настроек приложения
* */

//AppConfig настройки приложения
// VersionApp - версия приложения
// RootDir - корневая директория приложения
// IsServer - запуск приложения как сервер (true), как клиент (false)
// ToConnectServerHTTPS - параметры для соединения с сервером 'Master'
// LocalServerHTTPS - параметры настройки локального сервера
// DirectoryStoringProcessedFiles - директории для хранения обработанных файлов
// PathLogFiles - место расположение лог-файла приложения
// RefreshIntervalTelemetryInfo - интервал обновления системной информации
// TimeReconnect - актуально только в режиме isServer = false, тогда с заданным интервалом времени будут попытки соединения с адресом мастера
// MaxSizeTransferredChunkFile - максимальный размер передаваемого кусочка файла
// ForLocalUse - устанавливается в true если планируется осуществлять взаимодействие с приложением ещё и через Unix сокет
// NameUnixSocket - название файла используемого как Unix сокет
type AppConfig struct {
	VersionApp                     string
	RootDir                        string
	IsServer                       bool
	ToConnectServerHTTPS           []settingsToConnectServerHTTPS
	LocalServerHTTPS               settingsLocalServerHTTPS
	DirectoryStoringProcessedFiles settingsDirectoryStoreFiles
	PathLogFiles                   string
	RefreshIntervalTelemetryInfo   int
	TimeReconnect                  int
	MaxSizeTransferredChunkFile    int
	ForLocalUse                    bool
	NameUnixSocket                 string
}

//settingsToConnectServerHTTPS настройки сервера с которым устанавливается подключение в режиме клиента
type settingsToConnectServerHTTPS struct {
	IP    string
	Port  int
	Token string
}

//settingsLocalServerHTTPS настройки локального сервера HTTPS
type settingsLocalServerHTTPS struct {
	IP                  string
	Port                int
	AuthenticationToken string
	PathCertFile        string
	PathPrivateKeyFile  string
	PathRootCA          string
}

//settingsDirectoryStoreFiles настройки с путями к директориям для хранения файлов
type settingsDirectoryStoreFiles struct {
	Raw, Object string
}
