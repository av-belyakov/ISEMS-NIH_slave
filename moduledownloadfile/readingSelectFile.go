package moduledownloadfile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//ReadingFileParameters параметры для функции 'readingFile'
// TaskID - ID задачи
// ClientID - ID клиента для которого выполянется задача
// FileName - имя файла
// PathDirName - путь до директории с файлами
// StrHex - хеш строка вида '1:<task ID>:<file hex>' (примняется для идентификации кусочков файла)
// MaxChunkSize - максимальный размер передаваемой части файла
// NumReadCycle - кол-во передоваемых частей
// ChanCWTResBinary - канал для передачи бинарных данных
// ChanCWTResText - канал для передачи текстовых данных
type ReadingFileParameters struct {
	TaskID           string
	ClientID         string
	FileName         string
	PathDirName      string
	StrHex           string
	MaxChunkSize     int
	NumReadCycle     int
	ChanCWTResBinary chan<- configure.MsgWsTransmission
	ChanCWTResText   chan<- configure.MsgWsTransmission
}

//ReadingFile осуществляет чтение бинарного файла побайтно и передачу прочитанных байт в канал
func ReadingFile(rfp ReadingFileParameters, saveMessageApp *savemessageapp.PathDirLocationLogFiles, chanStop <-chan struct{}) error {
	fmt.Println("START func 'ReadingFile'...")

	file, err := os.Open(path.Join(rfp.PathDirName, rfp.FileName))
	if err != nil {
		return err
	}
	defer file.Close()

	chunkSize := (rfp.MaxChunkSize - len(rfp.StrHex))

	fmt.Printf("START func 'ReadingFile', chunk size = %v\n", chunkSize)

	var fileIsReaded error
DONE:
	for i := 0; i <= rfp.NumReadCycle; i++ {
		bytesTransmitted := []byte(rfp.StrHex)

		if i == 0 {
			fmt.Printf("\tReader byteTransmitted = %v\n", len(bytesTransmitted))
		}

		if fileIsReaded == io.EOF {
			break DONE
		}

		select {
		case <-chanStop:
			break DONE

		default:
			data, err := readNextBytes(file, chunkSize, i)
			if err != nil {
				if err != io.EOF {
					if i == 0 {
						fmt.Printf("func 'ReadingFile', ERROR %v\n", fmt.Sprint(err))
					}

					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

					break DONE
				}

				fileIsReaded = io.EOF
			}

			bytesTransmitted = append(bytesTransmitted, data...)

			if i == 0 || i == 1 {
				fmt.Printf("Reader chunk = %v, DATA = %v\n", len(bytesTransmitted), len(data))
				fmt.Printf("func 'ReadingFile', send next byte... %v\n", bytesTransmitted[:67])
			}

			rfp.ChanCWTResBinary <- configure.MsgWsTransmission{
				ClientID: rfp.ClientID,
				Data:     &data,
			}

		}
	}

	//сообщение об успешном останове задачи (сообщение об успешном завершении передачи файла не передается,
	// master сам решает когда файл передан полностью основываясь на количестве переданных частей и общем
	// размере файла)
	if fileIsReaded == nil {
		msgResJSON, err := json.Marshal(configure.MsgTypeDownloadControl{
			MsgType: "download control",
			Info: configure.DetailInfoMsgDownload{
				TaskID:  rfp.TaskID,
				Command: "file transfer stopped",
			},
		})
		if err != nil {
			return err
		}

		rfp.ChanCWTResText <- configure.MsgWsTransmission{
			ClientID: rfp.ClientID,
			Data:     &msgResJSON,
		}
	}

	return nil
}

func readNextBytes(file *os.File, number, nextNum int) ([]byte, error) {
	bytes := make([]byte, number)
	var off int64

	if nextNum != 0 {
		off = int64(number * nextNum)
	}

	rb, err := file.ReadAt(bytes, off)
	if err != nil {
		if err == io.EOF {
			return bytes[:rb], err
		}

		return nil, err
	}

	return bytes, nil
}
