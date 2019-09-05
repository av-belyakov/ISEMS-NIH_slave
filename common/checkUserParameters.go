package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ISEMS-NIH_slave/configure"
)

//CheckParametersFiltration проверяет параметры фильтрации
func CheckParametersFiltration(fccpf *configure.FiltrationControlCommonParametersFiltration) (string, bool) {
	//проверяем наличие ID источника
	if fccpf.ID == 0 {
		return "Отсутствует идентификатор источника. Задача отклонена.", false
	}

	//проверяем временной интервал
	isZero := ((fccpf.DateTime.Start == 0) || (fccpf.DateTime.End == 0))
	if isZero || (fccpf.DateTime.Start > fccpf.DateTime.End) {
		return "Задан неверный временной интервал. Задача отклонена.", false
	}

	//проверяем тип протокола
	if strings.EqualFold(fccpf.Protocol, "") {
		fccpf.Protocol = "any"
	}

	isProtoTCP := strings.EqualFold(fccpf.Protocol, "tcp")
	isProtoUDP := strings.EqualFold(fccpf.Protocol, "udp")
	isProtoANY := strings.EqualFold(fccpf.Protocol, "any")

	if !isProtoTCP && !isProtoUDP && !isProtoANY {
		return "Задан неверный идентификатор транспортного протокола. Задача отклонена.", false
	}

	isEmpty := true

	circle := func(fp map[string]map[string]*[]string, f func(string, *[]string) error) error {
		for pn, pv := range fp {
			var err error

			for _, v := range pv {
				if err = f(pn, v); err != nil {
					return err
				}
			}

			if err != nil {
				return err
			}
		}

		return nil
	}

	checkIPOrPortOrNetwork := func(paramType string, param *[]string) error {
		changeIP := func(item string) bool {
			ok, _ := CheckStringIP(item)

			return ok
		}

		changePort := func(item string) bool {
			p, err := strconv.Atoi(item)
			if err != nil {
				return false
			}

			if p == 0 || p > 65536 {
				return false
			}

			return true
		}

		changeNetwork := func(item string) bool {
			ok, _ := CheckStringNetwork(item)

			return ok
		}

		iteration := func(param *[]string, f func(string) bool) bool {
			if len(*param) == 0 {
				return true
			}

			isEmpty = false

			for _, v := range *param {
				if ok := f(v); !ok {
					return false
				}
			}

			return true
		}

		switch paramType {
		case "IP":
			if ok := iteration(param, changeIP); !ok {
				return errors.New("Неверные параметры фильтрации, один или более переданных пользователем IP адресов имеет некорректное значение")
			}

		case "Port":
			if ok := iteration(param, changePort); !ok {
				return errors.New("Неверные параметры фильтрации, один или более из заданных пользователем портов имеет некорректное значение")
			}

		case "Network":
			if ok := iteration(param, changeNetwork); !ok {
				return errors.New("Неверные параметры фильтрации, некорректное значение маски подсети заданное пользователем")
			}

		}

		return nil
	}

	filterParameters := map[string]map[string]*[]string{
		"IP": map[string]*[]string{
			"Any": &fccpf.Filters.IP.Any,
			"Src": &fccpf.Filters.IP.Src,
			"Dst": &fccpf.Filters.IP.Dst,
		},
		"Port": map[string]*[]string{
			"Any": &fccpf.Filters.Port.Any,
			"Src": &fccpf.Filters.Port.Src,
			"Dst": &fccpf.Filters.Port.Dst,
		},
		"Network": map[string]*[]string{
			"Any": &fccpf.Filters.Network.Any,
			"Src": &fccpf.Filters.Network.Src,
			"Dst": &fccpf.Filters.Network.Dst,
		},
	}

	//проверка ip адресов, портов и подсетей
	if err := circle(filterParameters, checkIPOrPortOrNetwork); err != nil {
		return fmt.Sprintf("%v. Задача отклонена.", err), false
	}

	//проверяем параметры свойства 'Filters' на пустоту
	if isEmpty {
		return "Невозможно начать фильтрацию, необходимо указать хотябы один искомый ip адрес, порт или подсеть. Задача отклонена.", false
	}

	return "", true
}

//CheckParametersDownloadFile проверяем параметры запроса на скачивание файла
func CheckParametersDownloadFile(dimd configure.DetailInfoMsgDownload) (string, bool) {
	if len(dimd.TaskID) == 0 {
		return fmt.Sprint("Невозможно начать выгрузку файла, ID задачи не задан."), false
	}

	if len(dimd.PathDirStorage) == 0 {
		return fmt.Sprint("Невозможно начать выгрузку файла, не задана директория в которой хранится требуемый файл."), false
	}

	if len(dimd.FileOptions.Name) == 0 {
		return fmt.Sprint("Невозможно начать выгрузку файла, не задано имя файла."), false
	}

	if len(dimd.FileOptions.Hex) == 0 {
		return fmt.Sprint("Невозможно начать выгрузку файла, не указана хеш-сумма запрашиваемого файла."), false
	}

	return "", true
}
