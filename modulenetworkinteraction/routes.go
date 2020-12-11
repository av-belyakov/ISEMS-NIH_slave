package modulenetworkinteraction

/*
* Маршрутизация запросов поступающих через протокол websocket
* */

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulenetworkinteraction/handlers"
	"ISEMS-NIH_slave/savemessageapp"
)

type msgTypeJSON struct {
	MsgType string `json:"messageType"`
}

//RouteWssConnect маршрутизация запросов
func RouteWssConnect(
	cwtResText chan<- configure.MsgWsTransmission,
	cwtResBinary chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	sma *configure.StoreMemoryApplication,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles,
	cwtReq <-chan configure.MsgWsTransmission) {

	var mtJSON msgTypeJSON

	for msg := range cwtReq {
		if err := json.Unmarshal(*msg.Data, &mtJSON); err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    "RouteWssConnect",
			})
		}

		fmt.Printf("func 'RouteWssConnect', received messge '%v'\n", mtJSON)

		switch mtJSON.MsgType {
		case "ping":
			go handlers.HandlerMessageTypePing(sma, msg.Data, msg.ClientID, appc, saveMessageApp, cwtResText)

		case "filtration":
			handlers.HandlerMessageTypeFiltration(sma, msg.Data, msg.ClientID, appc.DirectoryStoringProcessedFiles.Raw, saveMessageApp, cwtResText)

		case "download files":
			handlers.HandlerMessageTypeDownload(sma, msg.Data, msg.ClientID, appc, saveMessageApp, cwtResText, cwtResBinary)

		case "telemetry":
			go handlers.HandlerMessageTypeTelemetry(sma, msg.Data, msg.ClientID, saveMessageApp, cwtResText)
		}
	}
}
