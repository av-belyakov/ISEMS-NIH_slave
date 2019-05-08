package modulenetworkinteraction

/*
* Маршрутизация запросов поступающих через протокол websocket
*
* Версия 0.1, дата релиза 02.04.2019
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
	cwtReq <-chan configure.MsgWsTransmission) {

	fmt.Println("START function 'RouteWssConnect'...")

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	var mtJSON msgTypeJSON

	for msg := range cwtReq {
		//		fmt.Printf("Client ID:%v, request data count: %v\n", msg.ClientID, msg.Data)

		if err := json.Unmarshal(*msg.Data, &mtJSON); err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
		}

		switch mtJSON.MsgType {
		case "ping":
			fmt.Println("--- resived message JSON 'PING', func 'RouteWssConnect'")
			fmt.Printf("client ID %v\n", msg.ClientID)

			go handlers.HandlerMessageTypePing(cwtResText, sma, msg.Data, msg.ClientID)

		case "filtration":
			fmt.Println("--- resived message JSON 'FILTRATION', func 'RouteWssConnect'")
			fmt.Printf("client ID %v\n", msg.ClientID)

			/*
				запрос на фильтрацию доходит,
				теперь его нужно собирать если он состоит из нескольких частей
				(собирать как в moth_go) и обрабатывать
			*/
			go handlers.HandlerMessageTypeFiltration(cwtResText, sma, msg.Data, msg.ClientID)

		case "download files":
			fmt.Println("--- resived message JSON 'DOWNLOAD FILES', func 'RouteWssConnect'")
			fmt.Printf("client ID %v\n", msg.ClientID)

		}
	}
}
