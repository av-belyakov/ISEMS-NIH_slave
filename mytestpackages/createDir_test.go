package mytestpackages_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	//. "github.com/onsi/gomega"
	//. "ISEMS-NIH_master/mytestpackages"
)

var _ = Describe("Test Create Directory", func() {
	Context("Тест 1: Создание директорий если их нет", func() {
		listDir := []string{"/home/ISEMS_NIH_slave/ISEMS_NIH_slave_RAW", "/home/ISEMS_NIH_slave/ISEMS_NIH_slave_OBJECT"}

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

			pathDir := fmt.Sprintf("/%v/%v", rootDir, createDir)
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
				fmt.Println("11111")

				if err := mkDirectory("/", dir); err != nil {
					fmt.Println(err)
				}

				break
			}

			list := strings.Split(dir, "/")

			fmt.Printf("%q", list)
			fmt.Println("222222")

			for i := 0; i < len(list)-1; i++ {
				rd := list[i]
				if i != 0 {
					rd = strings.Join(list[:i+1], "/")
				}

				if err := mkDirectory(rd, list[i+1]); err != nil {
					fmt.Println(i)
					fmt.Println(err)
				}
			}

			fmt.Printf("COUNT list directory: %v\n", len(list))

		}
	})
})
