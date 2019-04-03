package configure

/*
* Типы JSON сообщений принимаемых от клиента
* */

//DetailInfoMsgPingPong подробная информация
type DetailInfoMsgPingPong struct {
	MaxCountProcessFiltration int8 `json:"maxCountProcessFiltration"`
	EnableTelemetry           bool `json:"enableTelemetry"`
}

//MsgTypePingPong сообщение типа ping
type MsgTypePingPong struct {
	MsgType string                `json:"messageType"`
	Info    DetailInfoMsgPingPong `json:"info"`
}
