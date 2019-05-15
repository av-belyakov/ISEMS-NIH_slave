package modulefiltrationfile

/*
* Модуль выполняющий фильтрацию файлов сетевого трафика
*
* Версия 0.1, дата релиза 15.05.2019
* */

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//ProcessingFiltration выполняет фильтрацию сет. трафика
func ProcessingFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID, directoryStoringProcessedFiles string) {

	fmt.Println("START function 'ProcessingFiltration'...")

	saveMessageApp := savemessageapp.New()
	errMsg := configure.MsgTypeError{
		MsgType: "error",
		Info: configure.DetailInfoMsgError{
			TaskID: taskID,
		},
	}

	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		errMsg.Info.ErrorName = "invalid value received"
		errMsg.Info.ErrorDescription = "Невозможно начать выполнение фильтрации, принят некорректный идентификатор клиента или задачи"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

		return
	}

	fmt.Printf("Параметры задачи по фильтрации:\n%v\n", info)

	//создаем директорию для хранения отфильтрованных файлов
	if err := createDirectoryForFiltering(info.DateTimeStart, taskID, directoryStoringProcessedFiles); err != nil {
		errMsg.Info.ErrorName = "unable to create directory"
		errMsg.Info.ErrorDescription = "Ошибка при создании директории для хранения отфильтрованных файлов"

		msgJSON, err := json.Marshal(errMsg)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}

		cwtResText <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &msgJSON,
		}

		_ = saveMessageApp.LogMessage("error", fmt.Sprintf("incorrect parameters for filtering (client ID: %v, task ID: %v)", clientID, taskID))

		return
	}

	//создаем файл README с описанием параметров фильтрации

	//если не используются индексы строим список файлов
	if !info.UseIndex {

	}

	//инициализируем выполнение фильтрации
	executeFiltration(cwtResText, sma, clientID, taskID)

	/*
		отправка сообщений в канал, для дольнейшей передачи ISEMS-NIH_master
		cwtResText <- configure.MsgWsTransmission{
					ClientID: clientID,
					Data:     &msgJSON,
				}
	*/
}

func createDirectoryForFiltering(dts int, taskID, dspf string) (string, error) {
	dateTimeStart := time.Unix(int64(dts), 0)

	dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + taskID
	filePath := path.Join(dspf, "/", dirName)

	return filePath, os.MkdirAll(filePath, 0766)
	/*prf *configure.ParametrsFunctionRequestFilter, mft *configure.MessageTypeFilter, ift *configure.InformationFilteringTask) error {
	dateTimeStart := time.Unix(int64(mft.Info.Settings.DateTimeStart), 0)

	dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + mft.Info.TaskIndex

	filePath := path.Join(prf.PathStorageFilterFiles, "/", dirName)

	ift.TaskID[mft.Info.TaskIndex].DirectoryFiltering = filePath

	return os.MkdirAll(filePath, 0766)*/
}

//выполнение фильтрации
func executeFiltration(
	cwtResText chan<- configure.MsgWsTransmission,
	sma *configure.StoreMemoryApplication,
	clientID, taskID string) {

}
