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
type DetailInfoMsgFiltration struct{}

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

//MsgTypeInformation информационное сообщение
type MsgTypeInformation struct {
	MsgType string             `json:"messageType"`
	Info    InformationMessage `json:"info"`
}

//InformationMessage информационное сообщение, подробная информация
type InformationMessage struct {
	TaskID string
}
