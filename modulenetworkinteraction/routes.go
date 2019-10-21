package modulenetworkinteraction

/*
* Маршрутизация запросов поступающих через протокол websocket
*
* Версия 0.2, дата релиза 05.09.2019
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
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		switch mtJSON.MsgType {
		case "ping":
			go handlers.HandlerMessageTypePing(sma, msg.Data, msg.ClientID, saveMessageApp, cwtResText)

		case "filtration":
			handlers.HandlerMessageTypeFiltration(sma, msg.Data, msg.ClientID, appc.DirectoryStoringProcessedFiles.Raw, saveMessageApp, cwtResText)

		case "download files":
			fmt.Printf("\t func 'RouteWssConnect' resived message JSON 'DOWNLOAD FILES', func 'RouteWssConnect', client ID %v\n", msg.ClientID)

			handlers.HandlerMessageTypeDownload(sma, msg.Data, msg.ClientID, appc, saveMessageApp, cwtResText, cwtResBinary)
		}
	}
}
