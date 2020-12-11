package handlers

import (
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeTelemetry обработчик сообщений типа 'telemetry'
func HandlerMessageTypeTelemetry(
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtResText chan<- configure.MsgWsTransmission) {

	fn := "HandlerMessageTypeTelemetry"

	fmt.Printf("func '%v', START...", fn)
}
