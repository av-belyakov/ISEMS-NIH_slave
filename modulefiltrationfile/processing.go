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

	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Параметры задачи по фильтрации:\n%v\n", info)

	/*
		отправка сообщений в канал, для дольнейшей передачи ISEMS-NIH_master
		cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &msgJSON,
				}
	*/
}
