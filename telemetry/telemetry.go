package telemetry

/*
* Модуль отвечающий за передачу телеметрии
*
* Версия 0.1, дата релиза 04.04.2019
* */

import (
	"fmt"
	"time"

	"ISEMS-NIH_slave/configure"
)

//TransmissionTelemetry передача телеметрии
func TransmissionTelemetry(
	cwtResText chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	sma *configure.StoreMemoryApplication) {

	fmt.Println("START function 'Telemetry'...")

	ticker := time.NewTicker(time.Duration(appc.RefreshIntervalTelemetryInfo) * time.Second)
	for range ticker.C {
		settingsAllClient := sma.GetAllClientSettings()

		fmt.Printf("ALL LIST CLIENTS: %v\n", settingsAllClient)

		count := len(settingsAllClient)
		if count == 0 {
			continue
		}

		cl := make([]string, 0, count)
		for clientID, s := range settingsAllClient {

			fmt.Printf("clientID %v, telemetry: %v\n", clientID, s.SendsTelemetry)

			if s.SendsTelemetry {
				cl = append(cl, clientID)
			}
		}

		fmt.Printf("CLIENT LIST send telemetry %v\n", cl)

		if len(cl) == 0 {
			continue
		}

		go GetSystemInformation(cwtResText, cl, sma)
	}
}
