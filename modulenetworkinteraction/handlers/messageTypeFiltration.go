package handlers

import (
	"ISEMS-NIH_slave/configure"
	"fmt"
)

//HandlerMessageTypeFiltration обработчик сообщений типа 'Filtration'
func HandlerMessageTypeFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string) {

	fmt.Println("START function 'HandlerMessageTypeFiltration'...")

}
