package mytestpackages_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/modulefiltrationfile"
)

var _ = Describe("Function Test", func() {
	//создаем новый репозиторий для хранения информации, в том числе о задачах
	sma := configure.NewRepositorySMA()

	//генерируем хеш для clientID
	clientID := common.GetUniqIDFormatMD5("client id")
	sma.SetClientSetting(clientID, configure.ClientSettings{
		ConnectionStatus: true,
		IP:               "36.100.0.71",
		Port:             "145",
		Token:            "ds9929jd99h923h9h39h9hf3f",
	})

	Context("Тест 1: Проверяем наличие выполняемых задач", func() {
		It("Количество выполняемых задач должно быть равно 0", func() {
			lt, _ := sma.GetListTasksFiltration(clientID)
			Expect(len(lt)).To(Equal(0))
		})
	})

	Context("Тест 2: Добавляем задачу для нового пользователя", func() {
		It("Количество выполняемых задач должно быть равно 1", func() {
			taskID := common.GetUniqIDFormatMD5("task id")

			sma.AddTaskFiltration(clientID, taskID, &configure.FiltrationTasks{
				DateTimeStart: 1557846213,
				DateTimeEnd:   1557848619,
				Protocol:      "tcp",
			})

			lt, _ := sma.GetListTasksFiltration(clientID)
			Expect(len(lt)).To(Equal(1))
		})
	})

	Context("Тест 3: Добавляем еще 3 задачи", func() {
		It("Так как количество задач превышает лимит в 3 задачи, вывести сообщение об ошибке", func() {
			for i := 0; i < 3; i++ {
				num := int64(i + 1*100)

				taskID := common.GetUniqIDFormatMD5("task id" + string(i))

				sma.AddTaskFiltration(clientID, taskID, &configure.FiltrationTasks{
					DateTimeStart: 1557846213 + num,
					DateTimeEnd:   1557848619 + num,
					Protocol:      "tcp",
				})
			}

			lt, _ := sma.GetListTasksFiltration(clientID)

			var isExist bool
			if len(lt) >= 3 {
				isExist = true
			}

			Expect(isExist).To(Equal(true))
		})
	})

	Context("Тест 4: Проверка параметорв фильтрации", func() {
		It("Должно выернуть 'TRUE' если все параметры фильтрации валидны", func() {
			_, ok := common.CheckParametersFiltration(&configure.FiltrationControlCommonParametersFiltration{
				ID: 11000,
				DateTime: configure.DateTimeParameters{
					Start: 1557867813,
					End:   1557887814,
				},
				Protocol: "any",
				Filters: configure.FiltrationControlParametersNetworkFilters{
					IP: configure.FiltrationControlIPorNetorPortParameters{
						Any: []string{"134.56.34.23", "87.56.100.34"},
					},
					Port: configure.FiltrationControlIPorNetorPortParameters{
						Dst: []string{"80", "23111", "161"},
					},
					Network: configure.FiltrationControlIPorNetorPortParameters{
						Src: []string{"78.100.23.56/24"},
					},
				},
			})

			//fmt.Printf("***** Error message: %v****\n", msg)

			Expect(ok).Should(Equal(true))
		})
	})

	lf := map[string][]string{
		"D_1": []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9"},
		"D_2": []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11"},
		"D_3": []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8"},
		"D_4": []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10"},
		"D_5": []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12", "f13"},
	}

	sizeChunk := 4

	lft := map[string]int{}
	for d, f := range lf {
		lft[d] = len(f)
	}

	countParts := common.GetCountPartsMessage(lft, sizeChunk)

	Context("Тест 5: Составление списка файлов найденных в результате поиска по индексам", func() {
		It("Должен получится список состоящий из 51 фрагментированного имени файлов", func() {
			var countFiles int
			for i := 1; i <= countParts; i++ {
				list := common.GetChunkListFiles(i, sizeChunk, countParts, lf)

				//fmt.Printf("List: %v\n", list)

				for _, v := range list {
					countFiles += len(v)
				}
			}

			//fmt.Printf("Test 5, ALL COUNT FILES = %v\n", countFiles)

			Expect(countFiles).Should(Equal(51))
		})
	})

	//генерируем хеш для clientID и taskID
	cID := common.GetUniqIDFormatMD5("client id1")
	tID := common.GetUniqIDFormatMD5("task id1")

	Context("Тест 6: Установка параметров клиента", func() {
		sma.SetClientSetting(cID, configure.ClientSettings{
			ConnectionStatus: true,
			IP:               "142.36.9.78",
			Port:             "13145",
			Token:            "ds9929jd99h923h9h39h9hf3f",
		})

		It("Возвращает 'TRUE' если клиент с указанным ID существует", func() {
			_, ok := sma.GetClientSetting(cID)

			Expect(ok).To(Equal(true))
		})
	})

	Context("Тест 7: Склейка списка файлов найденных в результате поиска по индексам", func() {
		var countFiles int
		var e error

		sma.AddTaskFiltration(cID, tID, &configure.FiltrationTasks{
			DateTimeStart: 1557856213,
			DateTimeEnd:   1557858619,
			Protocol:      "tcp",
		})

		for i := 1; i <= countParts; i++ {
			list := common.GetChunkListFiles(i, sizeChunk, countParts, lf)

			//fmt.Printf("List: %v\n", list)

			for _, v := range list {
				countFiles += len(v)
			}

			_, err := common.MergingFileListForTaskFiltration(sma, &configure.MsgTypeFiltrationControl{
				MsgType: "filtration",
				Info: configure.SettingsFiltrationControl{
					TaskID:                 tID,
					Command:                "start",
					IndexIsFound:           true,
					CountIndexFiles:        51,
					NumberMessagesFrom:     [2]int{i, countParts},
					ListFilesReceivedIndex: list,
				},
			}, cID)

			if err == nil {
				e = nil
			}
		}

		It("Ошибки при склейке быть не должно", func() {
			Expect(e).NotTo(HaveOccurred())
		})
	})

	Context("Тест 8: Вывести список переданных файлов, найденных в результате поиска по индексам", func() {
		ti, err := sma.GetInfoTaskFiltration(cID, tID)

		It("Ошибки при запросе информации о задаче быть не должно", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("Должен получится список состоящий из 51 файла", func() {
			var num int
			for _, v := range ti.ListFiles {
				num += len(v)
			}

			Expect(num).To(Equal(51))
		})
	})

	Context("Тест 9: Тестируем функцию 'GetChunkListFilesFound' которая делит отображение с информацией о файлах на сегменты", func() {
		listFileInfo, sizeFiles, err := modulefiltrationfile.GetListFoundFiles("/TRAFFIC_FILTER")
		numFiles := len(listFileInfo)
		const chunkSize = 10

		fmt.Printf("Num files: %v, common size all files = %v\n", numFiles, sizeFiles)

		numParts := common.CountNumberParts(numFiles, chunkSize)
		newList := make([]map[string]*configure.InputFilesInformation, 0, numParts)

		It("При получении списка файлов не должно быть ошибок", func() {
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("Получить список файлов ввиде отображения, по количеству равному 53", func() {
			Expect(numFiles).Should(Equal(55))
		})

		It("Получить количество частей сообщения равных 6", func() {
			Expect(numParts).Should(Equal(6))
		})

		It("Должно получится новое отображение заданной длинны, содержащее информацию о файлах", func() {
			for i := 1; i <= numParts; i++ {
				chunkListFiles := common.GetChunkListFilesFound(listFileInfo, i, numParts, chunkSize)

				newList = append(newList, chunkListFiles)
			}

			Expect(len(newList)).Should(Equal(6))
		})
	})
})
