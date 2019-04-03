package configure

/*
* Описание типа конфигурационных настроек приложения
*
* Версия 0.2, дата релиза 03.04.2019
* */

//AppConfig настройки приложения
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
