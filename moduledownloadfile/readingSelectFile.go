package moduledownloadfile

import (
	"io"
	"os"
	"path"

	"ISEMS-NIH_slave/configure"
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

//TypeChannelMsgRes описывает тип сообщения получаемого от обработчика файла
// ErrMsg - ошибка
// CauseStoped - причина останова
type TypeChannelMsgRes struct {
	ErrMsg      error
	CauseStoped string
}

//ReadingFile осуществляет чтение бинарного файла побайтно и передачу прочитанных байт в канал
func ReadingFile(chanRes chan<- TypeChannelMsgRes, rfp ReadingFileParameters, chanStop <-chan struct{}) {
	tcmr := TypeChannelMsgRes{
		CauseStoped: "completed",
	}

	file, err := os.Open(path.Join(rfp.PathDirName, rfp.FileName))
	if err != nil {
		tcmr.ErrMsg = err
		chanRes <- tcmr

		return
	}
	defer func() {
		//отправляем в канал пустое значение ошибки для удаления задачи
		chanRes <- tcmr

		close(chanRes)
		file.Close()
	}()

	chunkSize := (rfp.MaxChunkSize - len(rfp.StrHex))

	var fileIsReaded error
DONE:
	for i := 0; i <= rfp.NumReadCycle; i++ {
		bytesTransmitted := []byte(rfp.StrHex)

		if fileIsReaded == io.EOF {
			break DONE
		}

		select {
		case <-chanStop:
			tcmr.CauseStoped = "force stop"

			break DONE

		default:
			data, err := readNextBytes(file, chunkSize, i)
			if err != nil {
				if err != io.EOF {
					tcmr.ErrMsg = err

					break DONE
				}

				fileIsReaded = io.EOF
			}

			bytesTransmitted = append(bytesTransmitted, data...)

			rfp.ChanCWTResBinary <- configure.MsgWsTransmission{
				ClientID: rfp.ClientID,
				Data:     &bytesTransmitted,
			}

		}
	}
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
