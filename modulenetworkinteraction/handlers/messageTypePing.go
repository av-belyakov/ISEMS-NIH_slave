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
	smta *configure.StoreMemoryTasksApplication,
	req []byte,
	clientID string) {

	fmt.Println("START function 'HandlerMessageTypePing'...")

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	reqjson := configure.MsgTypePingPong{}
	if err := json.Unmarshal(req, &reqjson); err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

		return
	}

	mcpf := reqjson.Info.MaxCountProcessFiltration
	if mcpf == 0 {
		mcpf = 3
	}

	smta.SetSettingApplication(configure.SettingsApplication{
		MaxCountProcessFiltration: mcpf,
		EnableTelemetry:           reqjson.Info.EnableTelemetry,
	})

	resjson, err := json.Marshal(configure.MsgTypePingPong{
		MsgType: "pong",
		Info:    configure.DetailInfoMsgPingPong{},
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
