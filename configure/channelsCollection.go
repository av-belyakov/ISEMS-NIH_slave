package configure

/*
* Коллекция типов передоваемых через каналы
* */

//MsgWsTransmission передача и прием данных через websocket соединение
type MsgWsTransmission struct {
	ClientID string
	Data     *[]byte
}
