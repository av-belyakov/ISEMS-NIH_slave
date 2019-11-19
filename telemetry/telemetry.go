package telemetry

/*
* Модуль отвечающий за передачу телеметрии
* */

import (
	"time"

	"ISEMS-NIH_slave/configure"
)

//TransmissionTelemetry передача телеметрии
func TransmissionTelemetry(
	cwtResText chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	sma *configure.StoreMemoryApplication) {

	ticker := time.NewTicker(time.Duration(appc.RefreshIntervalTelemetryInfo) * time.Second)
	for range ticker.C {
		settingsAllClient := sma.GetAllClientSettings()

		count := len(settingsAllClient)
		if count == 0 {
			continue
		}

		cl := make([]string, 0, count)
		for clientID, s := range settingsAllClient {
			if s.SendsTelemetry {
				cl = append(cl, clientID)
			}
		}

		if len(cl) == 0 {
			continue
		}

		go GetSystemInformation(cwtResText, cl, sma)
	}
}
