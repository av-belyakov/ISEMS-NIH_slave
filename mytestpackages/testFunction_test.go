package mytestpackages_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"ISEMS-NIH_slave/common"
	"ISEMS-NIH_slave/configure"
	//. "ISEMS-NIH_master/mytestpackages"
)

var _ = Describe("Function Test", func() {
	//создаем новый репозиторий для хранения информации, в том числе о задачах
	sma := configure.NewRepositorySMA()

	//генерируем хеш для clientID и taskID
	clientID := common.GetUniqIDFormatMD5("client id")
	taskID := common.GetUniqIDFormatMD5("task id")

	Context("Тест 1: Проверяем наличие выполняемых задач", func() {
		It("Количество выполняемых задач должно быть равно 0", func() {
			lt, _ := sma.GetListTasksFiltration(clientID)
			Expect(len(lt)).To(Equal(0))
		})
	})

	Context("Тест 2: Добавляем задачу для нового пользователя", func() {
		It("Количество выполняемых задач должно быть равно 1", func() {
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

				//генерируем хеш для clientID и taskID
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
			msg, ok := common.CheckParametersFiltration(&configure.FiltrationControlCommonParametersFiltration{
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

			fmt.Printf("***** Error message: %v****\n", msg)

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

				fmt.Printf("List: %v\n", list)

				for _, v := range list {
					countFiles += len(v)
				}
			}

			fmt.Printf("Test 5, ALL COUNT FILES = %v\n", countFiles)

			Expect(countFiles).Should(Equal(51))
		})
	})

	Context("Тест 6: Склейка списка файлов найденных в результате поиска по индексам", func() {
		It("При успешном завершении объединении списков файлов получаем 'TRUE'", func() {
			var countFiles int
			for i := 1; i <= countParts; i++ {
				list := common.GetChunkListFiles(i, sizeChunk, countParts, lf)

				fmt.Printf("List: %v\n", list)

				for _, v := range list {
					countFiles += len(v)
				}

				layoutListCompleted, err := common.MergingFileListForTaskFiltration(sma, &configure.MsgTypeFiltrationControl{
					MsgType: "filtration",
					Info: configure.SettingsFiltrationControl{
						TaskID: taskID,
						Command: "start",
						IndexIsFound: true,
						CountIndexFiles: 51,
						NumberMessagesFrom: [2]{i, countParts},
						/*
						ДОДЕЛАТЬ ТЕСТ СО СКЛЕЙКОЙ СПИСКА ФАЙЛОВ
						*/
					},
				}, clientID, taskID)
			}

			fmt.Printf("Test 6, ALL COUNT FILES = %v\n", countFiles)
		})
	})
})
