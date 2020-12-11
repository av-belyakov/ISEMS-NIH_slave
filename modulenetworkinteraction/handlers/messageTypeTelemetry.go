package handlers

import (
	"encoding/json"
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

	fmt.Printf("func '%v', START...\n", fn)

	mttrJSON := configure.MsgTypeTelemetryRequest{}

	if err := json.Unmarshal(*req, &mttrJSON); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	fmt.Printf("received reguest '%v'\n", mttrJSON)

	/*
	   Сюда доходит запрос на получения телеметрии с ISEMS-NIH_master
	   Теперь нужно выполнить обработку этого запроса и отправить модулю ISEMS-NIH_master

	   В модуле ISEMS-NIH_master нужно обработать полученный ответ от конкретного источника,
	   удалить его из списка StoreMemoryTasks и отправить клиенту API

	   при этом если в списке источников StoreMemoryTasks будет пусто то изменить статус
	   задачи на 'выполненная', для последующего удаления задачи из StoreMemoryTasks
	*/

}
