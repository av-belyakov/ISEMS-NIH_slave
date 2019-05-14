package handlers

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypeFiltration обработчик сообщений типа 'Filtration'
func HandlerMessageTypeFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string) {

	fmt.Println("START function 'HandlerMessageTypeFiltration'...")

	saveMessageApp := savemessageapp.New()

	mtfcJSON := configure.MsgTypeFiltrationControl{}

	if err := json.Unmarshal(*req, &mtfcJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	if mtfcJSON.Info.Command == "start" {
		go StartFiltration(cwtResText, sma, &mtfcJSON, clientID, mtfcJSON.Info.TaskID)
	}

	if mtfcJSON.Info.Command == "stop" {

	}

}
