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
type DetailInfoMsgFiltration struct {
	TaskID string `json:"tid"`
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
