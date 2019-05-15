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

//HandlerMessageTypePing обработчик сообщений типа 'Ping'
func HandlerMessageTypePing(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	req *[]byte,
	clientID string) {

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	reqJSON := configure.MsgTypePing{}
	if err := json.Unmarshal(*req, &reqJSON); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	var mcpf int8 = 3
	if mcpf > 0 {
		mcpf = reqJSON.Info.MaxCountProcessFiltration
	}

	sma.SetApplicationSetting(configure.ApplicationSettings{
		StorageFolders: reqJSON.Info.StorageFolders,
	})

	cs, ok := sma.GetClientSetting(clientID)
	if !ok {
		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("unable to send message of type 'pong' client with ID %v does not exist", clientID))

		return
	}

	cs.MaxCountProcessFiltration = mcpf
	cs.SendsTelemetry = reqJSON.Info.EnableTelemetry

	sma.SetClientSetting(clientID, cs)

	resJSON, err := json.Marshal(configure.MsgTypePing{
		MsgType: "pong",
		Info:    configure.DetailInfoMsgPing{},
	})
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	cwtResText <- configure.MsgWsTransmission{
		ClientID: clientID,
		Data:     &resJSON,
	}
}
