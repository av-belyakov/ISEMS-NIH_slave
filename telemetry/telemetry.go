package telemetry

/*
* Модуль отвечающий за передачу телеметрии
* */

import (
	"encoding/json"
	"fmt"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//TransmissionTelemetry передача телеметрии
func TransmissionTelemetry(
	cwtResText chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	sma *configure.StoreMemoryApplication,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	fn := "TransmissionTelemetry"
	chanSysInfo := make(chan SysInfo)
	getListClients := func() ([]string, error) {
		cl := []string{}
		err := fmt.Errorf("there are no clients waiting to receive telemetry")

		settingsAllClient := sma.GetAllClientSettings()

		count := len(settingsAllClient)
		if count == 0 {
			return cl, err
		}

		for clientID, s := range settingsAllClient {
			if s.SendsTelemetry {
				cl = append(cl, clientID)
			}
		}

		if len(cl) == 0 {
			return cl, err
		}

		return cl, nil
	}

	go func() {
		for si := range chanSysInfo {
			cl, err := getListClients()
			if err != nil {
				saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})

				continue
			}

			resJSON, err := json.Marshal(si)
			if err != nil {
				saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})

				continue
			}

			//рассылаем всем клиентам ожидающим телеметрию
			for _, clientID := range cl {
				cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &resJSON,
				}
			}
		}
	}()

	ticker := time.NewTicker(time.Duration(appc.RefreshIntervalTelemetryInfo) * time.Second)
	for range ticker.C {
		if _, err := getListClients(); err == nil {
			go GetSystemInformation(chanSysInfo, sma, saveMessageApp)
		}
	}
}
