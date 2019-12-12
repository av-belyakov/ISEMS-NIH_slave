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

func handlerReqUnixSocket(
	cwtReq chan<- configure.MsgWsTransmission,
	conn *net.Conn,
	sma *configure.StoreMemoryApplication,
	appc *configure.AppConfig,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	fn := "handlerReqUnixSocket"

	for {
		buf := make([]byte, 512)
		nr, err := (*conn).Read(buf)
		if err != nil {
			return
		}

		message := buf[0:nr]

		tusi := configure.TypeUnixSocketInteraction{}
		if err := json.Unmarshal(message, &tusi); err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		/* все сообщения приходящие через Unix Socket должны содержать поле ClientID */
		if tusi.ClientID == "" {
			msgErr := "the client ID is missing. the message is not correct."
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: msgErr,
				FuncName:    fn,
			})

			_, err = (*conn).Write([]byte(fmt.Sprintf("Error: %v", msgErr)))
			(*conn).Close()

			return
		}

		clientID := fmt.Sprintf("unix_socket:%v", tusi.ClientID)

		//проверяем наличие настроек пользователя
		_, err = sma.GetClientSetting(clientID)
		if err != nil {
			if tusi.Token != appc.ToConnectUnixSocket.Token {
				msgErr := "invalid token received"
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: msgErr,
					FuncName:    fn,
				})

				_, err = (*conn).Write([]byte(fmt.Sprintf("Error: %v", msgErr)))
				(*conn).Close()

				return
			}

			//добавляем информацию об авторизованном пользователе (здесь же создаются хранилища для отслеживания выполняемых задач)
			sma.SetClientSetting(clientID, configure.ClientSettings{
				IP:              "local",
				Port:            "unix_socket",
				Token:           appc.ToConnectUnixSocket.Token,
				AccessIsAllowed: true,
			})

			//изменяем статус соединения
			if err := sma.ChangeSourceConnectionStatus(clientID, true); err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}
		}

		//добавляем дескриптор соединения для данного пользователя
		sma.AddLinkUnixSocketConnect(clientID, conn)

		cwtReq <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &message,
		}
	}
}

//UnixSocketInteraction модуль взаимодействия через Unix сокет
func UnixSocketInteraction(
	cwtReq chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	sma *configure.StoreMemoryApplication,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	fmt.Println("func 'UnixSocketInteraction' START...")

	fn := "UnixSocketInteraction"
	pathSocket := path.Join("/tmp", appc.ToConnectUnixSocket.SocketName)

	_ = os.RemoveAll(pathSocket)

	l, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: pathSocket,
	})
	if err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})
		}

		log.Printf("func 'UnixSocketInteraction' CONNECT: '%v'\n", conn.RemoteAddr().Network())

		go handlerReqUnixSocket(cwtReq, &conn, sma, appc, saveMessageApp)
	}
}
