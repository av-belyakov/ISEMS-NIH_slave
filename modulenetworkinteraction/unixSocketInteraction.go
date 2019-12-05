package modulenetworkinteraction

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

func handlerReqUnixSocket(conn net.Conn, appc *configure.AppConfig, saveMessageApp *savemessageapp.PathDirLocationLogFiles) {
	funcName := "handlerReqUnixSocket"

	for {
		buf := make([]byte, 512)
		nr, err := conn.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]

		tusi := configure.TypeUnixSocketInteraction{}
		if err := json.Unmarshal(data, &tusi); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    funcName,
			})
		}

		//проверяем токен
		if tusi.Token != appc.ToConnectUnixSocket.Token {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: "invalid token received",
				FuncName:    funcName,
			})

			_, err = conn.Write([]byte("Error: invalid token received"))

			return
		}

		//получаем параметры фильтрации
		mtfc := configure.MsgTypeFiltrationControl{}
		if err := json.Unmarshal(data, &mtfc); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    funcName,
			})
		}

		fmt.Printf("Unix socket client transmited - '%v'\n", mtfc)

		_, err = conn.Write([]byte("Token is valide, task START..."))
		if err != nil {
			fmt.Printf("Error writing socket: '%v'\n", err)
		}
	}
}

//UnixSocketInteraction модуль взаимодействия через Unix сокет
func UnixSocketInteraction(appc *configure.AppConfig, sma *configure.StoreMemoryApplication, saveMessageApp *savemessageapp.PathDirLocationLogFiles) {
	fmt.Println("func 'UnixSocketInteraction' START...")
	fmt.Printf("file Unix socket:'%v'\n", appc.ToConnectUnixSocket)

	funcName := "UnixSocketInteraction"
	pathSocket := path.Join("/tmp", appc.ToConnectUnixSocket.SocketName)

	_ = os.RemoveAll(pathSocket)

	l, err := net.Listen("unix", pathSocket)
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

		log.Printf("func 'UnixSocketInteraction' CONNECT: '%v'\n", conn.RemoteAddr().Network())

		go handlerReqUnixSocket(conn, appc, saveMessageApp)
	}
}
