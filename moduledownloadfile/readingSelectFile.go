package moduledownloadfile

import (
	"fmt"
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

//ReadingFile осуществляет чтение бинарного файла побайтно и передачу прочитанных байт в канал
func ReadingFile(chanError chan<- error, rfp ReadingFileParameters, chanStop <-chan struct{}) {
	fmt.Println("START func 'ReadingFile'...")

	file, err := os.Open(path.Join(rfp.PathDirName, rfp.FileName))
	if err != nil {
		chanError <- err

		return
	}
	defer func() {
		chanError <- nil

		close(chanError)
		file.Close()
	}()

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
			fmt.Printf("func 'ReadingFile', Resived message 'STOP', Value fileIsReaded equal '%v'\n", fileIsReaded)

			break DONE

		default:
			data, err := readNextBytes(file, chunkSize, i)
			if err != nil {
				if err != io.EOF {
					if i == 0 {
						fmt.Printf("func 'ReadingFile', ERROR %v\n", fmt.Sprint(err))
					}

					chanError <- err

					break DONE
				}

				fileIsReaded = io.EOF
			}

			bytesTransmitted = append(bytesTransmitted, data...)

			if (i == 0) || (i == 1) {
				fmt.Printf("Reader chunk = %v, DATA = %v\n", len(bytesTransmitted), len(data))
				fmt.Printf("func 'ReadingFile', send next byte... %v\n", bytesTransmitted[:67])
			}

			rfp.ChanCWTResBinary <- configure.MsgWsTransmission{
				ClientID: rfp.ClientID,
				Data:     &bytesTransmitted,
			}

		}
	}

	fmt.Println("func 'ReadingFile', COMPLITE CYCLE READING FILE")
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
