package handlers

/*
* Обработчик сообщений типа 'Ping'
*
* Версия 0.1, дата релиза 03.04.2019
* */

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerMessageTypePing обработчик сообщений типа
func HandlerMessageTypePing(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string) {

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	reqjson := configure.MsgTypePing{}
	if err := json.Unmarshal(*req, &reqjson); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	mcpf := reqjson.Info.MaxCountProcessFiltration
	if mcpf == 0 {
		mcpf = 3
	}

	sma.SetApplicationSetting(configure.ApplicationSettings{
		StorageFolders: reqjson.Info.StorageFolders,
	})

	cs, ok := sma.GetClientSetting(clientID)
	if !ok {
		_ = saveMessageApp.LogMessage("error", "unable to send message of type 'pong' client with ID "+clientID+" does not exist")

		return
	}

	cs.MaxCountProcessFiltration = mcpf
	cs.SendsTelemetry = reqjson.Info.EnableTelemetry

	sma.SetClientSetting(clientID, cs)

	resjson, err := json.Marshal(configure.MsgTypePing{
		MsgType: "pong",
		Info:    configure.DetailInfoMsgPing{},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	cwtResText <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &resjson,
	}
}
