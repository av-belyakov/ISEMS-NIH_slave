package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
*
* Версия 0.1, дата релиза 14.05.2019
* */

import (
	"fmt"

	"ISEMS-NIH_slave/configure"
)

//ProcessingFiltration выполняет фильтрацию сет. трафика
func ProcessingFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID string) {

	fmt.Println("START function 'ProcessingFiltration'...")
}
