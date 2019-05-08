package configure

/*
* Типы JSON сообщений принимаемых от клиента
* */

//DetailInfoMsgPing подробная информация
type DetailInfoMsgPing struct {
	MaxCountProcessFiltration int8     `json:"maxCountProcessFiltration"`
	EnableTelemetry           bool     `json:"enableTelemetry"`
	StorageFolders            []string `json:"storageFolders"`
}

//MsgTypePing сообщение типа ping
type MsgTypePing struct {
	MsgType string            `json:"messageType"`
	Info    DetailInfoMsgPing `json:"info"`
}
