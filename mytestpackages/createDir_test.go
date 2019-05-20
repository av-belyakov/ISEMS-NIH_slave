package mytestpackages_test

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
)

func createDirectory(listDir []string) error {
	mkDirectory := func(rootDir, createDir string) error {
		if rootDir == "" {
			return nil
		}

		files, err := ioutil.ReadDir(rootDir)
		if err != nil {
			return err
		}

		for _, fl := range files {
			if fl.Name() == createDir {
				return nil
			}
		}

		pathDir := path.Join(rootDir, createDir)
		if rootDir == "/" {
			pathDir = fmt.Sprintf("%v", createDir)
		}

		err = os.Mkdir(pathDir, 0777)
		if err != nil {
			return err
		}

		return nil
	}

	for _, dir := range listDir {
		if strings.Count(dir, "/") == 1 {
			if err := mkDirectory("/", dir); err != nil {
				return err
			}

			break
		}

		list := strings.Split(dir, "/")

		for i := 0; i < len(list)-1; i++ {
			rd := list[i]
			if i != 0 {
				rd = strings.Join(list[:i+1], "/")
			}

			if err := mkDirectory(rd, list[i+1]); err != nil {
				return err
			}
		}
	}

	return nil
}

func createDirectoryForFiltering(sma *configure.StoreMemoryApplication, clientID, taskID, dspf string) error {
	info, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	dateTimeStart := time.Unix(int64(info.DateTimeStart), 0)

	dirName := strconv.Itoa(dateTimeStart.Year()) + "_" + dateTimeStart.Month().String() + "_" + strconv.Itoa(dateTimeStart.Day()) + "_" + strconv.Itoa(dateTimeStart.Hour()) + "_" + strconv.Itoa(dateTimeStart.Minute()) + "_" + taskID
	filePath := path.Join(dspf, "/", dirName)

	if err := os.MkdirAll(filePath, 0766); err != nil {
		return err
	}

	if err := sma.SetInfoTaskFiltration(
		clientID,
		taskID,
		map[string]interface{}{
			"FileStorageDirectory": filePath,
		}); err != nil {
		return err
	}

	return nil
}

func createFileReadme(sma *configure.StoreMemoryApplication, clientID, taskID string) error {
	type FiltrationControlIPorNetorPortParameters struct {
		Any []string `xml:"any>value"`
		Src []string `xml:"src>value"`
		Dst []string `xml:"dst>value"`
	}

	type FilterSettings struct {
		Protocol      string                                   `xml:"filters>protocol"`
		DateTimeStart string                                   `xml:"filters>date_time_start"`
		DateTimeEnd   string                                   `xml:"filters>date_time_end"`
		IP            FiltrationControlIPorNetorPortParameters `xml:"filters>ip"`
		Port          FiltrationControlIPorNetorPortParameters `xml:"filters>port"`
		Network       FiltrationControlIPorNetorPortParameters `xml:"filters>network"`
	}

	type Information struct {
		XMLName            xml.Name `xml:"information"`
		DateTimeCreateTask string   `xml:"date_time_create_task"`
		FilterSettings
		UseIndex                bool `xml:"use_index"`
		CountIndexFiles         int  `xml:"count_index_files"`
		CountProcessedFiles     int  `xml:"count_processed_files"`
		CountNotFoundIndexFiles int  `xml:"count_not_found_index_files"`
	}

	task, err := sma.GetInfoTaskFiltration(clientID, taskID)
	if err != nil {
		return err
	}

	tct := time.Unix(int64(time.Now().Unix()), 0)
	dtct := strconv.Itoa(tct.Day()) + " " + tct.Month().String() + " " + strconv.Itoa(tct.Year()) + " " + strconv.Itoa(tct.Hour()) + ":" + strconv.Itoa(tct.Minute())

	i := Information{
		UseIndex:                task.UseIndex,
		DateTimeCreateTask:      dtct,
		CountIndexFiles:         task.CountIndexFiles,
		CountProcessedFiles:     task.CountProcessedFiles,
		CountNotFoundIndexFiles: task.CountNotFoundIndexFiles,
		FilterSettings: FilterSettings{
			Protocol:      task.Protocol,
			DateTimeStart: fmt.Sprint(time.Unix(int64(task.DateTimeStart), 0)),
			DateTimeEnd:   fmt.Sprint(time.Unix(int64(task.DateTimeEnd), 0)),
			IP: FiltrationControlIPorNetorPortParameters{
				Any: task.Filters.IP.Any,
				Src: task.Filters.IP.Src,
				Dst: task.Filters.IP.Dst,
			},
			Port: FiltrationControlIPorNetorPortParameters{
				Any: task.Filters.Port.Any,
				Src: task.Filters.Port.Src,
				Dst: task.Filters.Port.Dst,
			},
			Network: FiltrationControlIPorNetorPortParameters{
				Any: task.Filters.Network.Any,
				Src: task.Filters.Network.Src,
				Dst: task.Filters.Network.Dst,
			},
		},
	}

	output, err := xml.MarshalIndent(i, "  ", "    ")
	if err != nil {
		return err
	}

	//fmt.Printf("storageDir: %v, README.xml", task.FileStorageDirectory)

	f, err := os.OpenFile(path.Join(task.FileStorageDirectory, "README.xml"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(output); err != nil {
		return err
	}

	return nil
}

var _ = Describe("Test Create Directory", func() {
	listDir := []string{"/home/ISEMS_NIH_slave/ISEMS_NIH_slave_RAW", "/home/ISEMS_NIH_slave/ISEMS_NIH_slave_OBJECT"}

	//создаем новый репозиторий для хранения информации, в том числе о задачах
	sma := configure.NewRepositorySMA()

	Context("Тест 1: Создание основных директорий приложения", func() {

		err := createDirectory(listDir)
		It("При создании директорий ошибок не должно быть ошибок", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Тест 2: Проверка наличия основных директорий приложения", func() {
		It("Должна существовать директория '/home/ISEMS_NIH_slave/ISEMS_NIH_slave_RAW'", func() {
			isExist := true

			if _, err := os.Stat("/home/ISEMS_NIH_slave/ISEMS_NIH_slave_RAW"); err != nil {
				fmt.Println(err)

				isExist = false
			}

			Expect(isExist).To(Equal(true))
		})

		It("Должна существовать директория '/home/ISEMS_NIH_slave/ISEMS_NIH_slave_OBJECT'", func() {
			isExist := true

			if _, err := os.Stat("/home/ISEMS_NIH_slave/ISEMS_NIH_slave_OBJECT"); err != nil {
				fmt.Println(err)

				isExist = false
			}

			Expect(isExist).To(Equal(true))
		})
	})

	//генерируем хеш для clientID и taskID
	clientID := common.GetUniqIDFormatMD5("client id2")
	taskID := common.GetUniqIDFormatMD5("task id2")

	Context("Тест 3: Создание директории для хранения отфильтрованных файлов", func() {
		sma.SetClientSetting(clientID, configure.ClientSettings{
			ConnectionStatus: true,
			IP:               "36.100.0.71",
			Port:             "145",
			Token:            "ds9929jd99h923h9h39h9hf3f",
		})

		sma.AddTaskFiltration(clientID, taskID, &configure.FiltrationTasks{
			DateTimeStart: 1557866213,
			DateTimeEnd:   1557868619,
			Protocol:      "tcp",
			Filters: configure.FiltrationControlParametersNetworkFilters{
				IP: configure.FiltrationControlIPorNetorPortParameters{
					Any: []string{"123.45.12.30", "78.90.100.21"},
					Src: []string{"10.23.20.1"},
				},
				Port: configure.FiltrationControlIPorNetorPortParameters{
					Dst: []string{"344", "8080"},
				},
			},
		})

		It("Запись информации о выполняемой задаче должна быть успешной", func() {
			_, err := sma.GetInfoTaskFiltration(clientID, taskID)

			Expect(err).NotTo(HaveOccurred())
		})

		It("При создание директории не должно быть ошибок", func() {
			err := createDirectoryForFiltering(sma, clientID, taskID, "/home/ISEMS_NIH_slave/ISEMS_NIH_slave_RAW")

			Expect(err).NotTo(HaveOccurred())
		})

		It("Директория должна быть созданна", func() {
			info, err := sma.GetInfoTaskFiltration(clientID, taskID)
			Expect(err).NotTo(HaveOccurred())

			_, e := os.Stat(info.FileStorageDirectory)
			Expect(e).NotTo(HaveOccurred())
		})

		It("При создании README.xml файла не должно быть ошибок", func() {
			err := createFileReadme(sma, clientID, taskID)

			Expect(err).NotTo(HaveOccurred())
		})

		It("Файл README.xml должен быть успешно создан", func() {
			info, err := sma.GetInfoTaskFiltration(clientID, taskID)
			Expect(err).NotTo(HaveOccurred())

			_, e := os.Stat(info.FileStorageDirectory + "/README.xml")
			Expect(e).NotTo(HaveOccurred())
		})
	})

	/*Context("Тест 4: создание README.txt файла", func() {
		err := createFileReadme(sma, clientID, taskID, "/home/ISEMS_NIH_slave/ISEMS_NIH_slave_RAW")

		It("При создании README.txt файла не должно быть ошибок", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("Файл README должен быть успешно создан", func() {
			info, err := sma.GetInfoTaskFiltration(clientID, taskID)
			Expect(err).NotTo(HaveOccurred())

			_, e := os.Stat(info.FileStorageDirectory + "/README.txt")
			Expect(e).NotTo(HaveOccurred())
		})
	})*/
})
