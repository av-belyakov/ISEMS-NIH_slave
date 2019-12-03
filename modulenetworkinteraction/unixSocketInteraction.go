package modulenetworkinteraction

import (
	"fmt"
	"net"
	"os"
	"path"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//UnixSocketInteraction модуль взаимодействия через Unix сокет
func UnixSocketInteraction(appc *configure.AppConfig, sma *configure.StoreMemoryApplication, saveMessageApp *savemessageapp.PathDirLocationLogFiles) {
	fmt.Println("func 'UnixSocketInteraction' START...")

	fmt.Printf("file Unix socket:'%v'\n", appc.NameUnixSocket)

	funcName := "UnixSocketInteraction"

	_ = os.RemoveAll(path.Join("/tmp", appc.NameUnixSocket))

	l, err := net.Listen("unix", path.Join("/tmp", appc.NameUnixSocket))
	if err != nil {

		fmt.Printf("func 'UnixSocketInteraction', ERROR: %v\n", err)

		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    funcName,
		})

		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    funcName,
			})
		}

		fmt.Printf("func 'UnixSocketInteraction' CONNECT: '%v'\n", conn.RemoteAddr().Network())
	}
}
