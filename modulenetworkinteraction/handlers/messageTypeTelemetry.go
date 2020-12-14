package handlers

import (
	"encoding/json"
	"fmt"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
	"ISEMS-NIH_slave/telemetry"
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

	if mttrJSON.Info.Command == "give me telemetry" {
		chanSysInfo := make(chan telemetry.SysInfo)

		go telemetry.GetSystemInformation(chanSysInfo, sma, saveMessageApp)

		si := <-chanSysInfo

		si.TaskID = mttrJSON.Info.TaskID
		resJSON, err := json.Marshal(si)
		if err != nil {
			saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &resJSON,
		}
	}
}
