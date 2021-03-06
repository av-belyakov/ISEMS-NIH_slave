package modulenetworkinteraction

/*
* Модуль сетевого взаимодействия осуществляет прием или установление
* соединений с клиентами приложения
* */

import (
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gorilla/websocket"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
	"ISEMS-NIH_slave/telemetry"
)

type serverSetting struct {
	IP, Port, Token        string
	StoreMemoryApplication *configure.StoreMemoryApplication
	SaveMessageApp         *savemessageapp.PathDirLocationLogFiles
}

type serverWebsocketSetting struct {
	StoreMemoryApplication *configure.StoreMemoryApplication
	Cwt                    chan<- configure.MsgWsTransmission
	SaveMessageApp         *savemessageapp.PathDirLocationLogFiles
}

type clientSetting struct {
	ID, IP, Port           string
	StoreMemoryApplication *configure.StoreMemoryApplication
	SaveMessageApp         *savemessageapp.PathDirLocationLogFiles
	TLSConf                *tls.Config
	Cwt                    chan<- configure.MsgWsTransmission
}

func createClientID(str string) string {
	h := md5.New()
	io.WriteString(h, str)

	return hex.EncodeToString(h.Sum(nil))
}

//при разрыве соединения удаляет дескриптор соединения и изменяет статус клиента
func connClose(
	c *websocket.Conn,
	sma *configure.StoreMemoryApplication,
	clientID, clientIP, requester string,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	fmt.Printf("CLOSE WSS LINK WITH IP '%v'\n", clientIP)

	saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
		TypeMessage: "info",
		Description: fmt.Sprintf("close wss link with ip '%v'", clientIP),
		FuncName:    "connClose",
	})

	fn := "connClose"

	if c != nil {
		c.Close()
	}

	//изменение статуса соединения
	if err := sma.ChangeSourceConnectionStatus(clientID, false); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	//отправляем запросы на останов передачи файлов для данного клиента
	if err := sendMsgStopDownloadFiles(sma, clientID); err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	/*
	   А здесь похоже не надо удалять информацию о клиенте
	   по тому что задача теряется!!!
	*/

	/*if requester == "server" {
		//удаляем параметры подключения клиента
		sma.DeleteClientSetting(clientID)
	}*/

	//удаляем дескриптор соединения
	sma.DelLinkWebsocketConnection(clientIP)
}

func sendMsgStopDownloadFiles(sma *configure.StoreMemoryApplication, clientID string) error {
	ldt, err := sma.GetAllInfoTaskDownload(clientID)
	if err != nil {
		return err
	}

	for _, ti := range ldt {
		//если задача не была завершена автоматически по мере выполнения
		if !ti.IsTaskCompleted {
			if ti.ChanStopReadFile != nil {
				ti.ChanStopReadFile <- struct{}{}

				return nil
			}
		}
	}

	//удаляем все задачи для данного клиента
	return sma.DelAllTaskDownload(clientID)
}

//MainNetworkInteraction модуль сетевого взаимодействия
func MainNetworkInteraction(appc *configure.AppConfig, sma *configure.StoreMemoryApplication, saveMessageApp *savemessageapp.PathDirLocationLogFiles) {
	fn := "MainNetworkInteraction"

	//читаем сертификат CA для того что бы клиент доверял сертификату переданному сервером
	rootCA, err := ioutil.ReadFile(appc.LocalServerHTTPS.PathRootCA)
	if err != nil {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: "root certificate was not added to the pool",
			FuncName:    fn,
		})
	}

	//создаем новый пул доверенных центров серификации и добавляем в него корневой сертификат
	cp := x509.NewCertPool()
	if ok := cp.AppendCertsFromPEM(rootCA); !ok {
		saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	conf := &tls.Config{
		ServerName: "isems_nih_master",
		RootCAs:    cp,
	}
	conf.BuildNameToCertificate()

	//инициализируем канал для передачи текстовых данных через websocket соединение
	cwtResText := make(chan configure.MsgWsTransmission)
	//инициализируем канал для передачи бинарных данных через websocket соединение
	cwtResBinary := make(chan configure.MsgWsTransmission)
	//инициализируем канал для приема текстовых данных через websocket соединение
	cwtReq := make(chan configure.MsgWsTransmission)

	//обработка ответов поступающих изнутри приложения через канал cwtRes
	go func() {
		getConnLink := func(clientID string) (*configure.WssConnection, error) {
			s, err := sma.GetClientSetting(clientID)
			if err != nil {
				return nil, fmt.Errorf("the ip address cannot be found by the given client ID '%v'", clientID)
			}

			//получаем линк websocket соединения
			c, ok := sma.GetLinkWebsocketConnect(s.IP)
			if !ok {
				return nil, fmt.Errorf("no connection found at websocket link ip address '%v'", s.IP)
			}

			return c, nil
		}

		for {
			select {
			case msgText := <-cwtResText:
				//если запрос пришел через unix socket
				if strings.Contains(msgText.ClientID, "unix_socket") {
					if c, ok := sma.GetLinkUnixSocketConnect(msgText.ClientID); ok {
						if _, err := (*c.Link).Write(*msgText.Data); err != nil {
							sma.DelLinkUnixSocketConnection(msgText.ClientID)
						}
					}

					continue
				}

				//выполняется только если запрос пришел через WebSocket
				c, err := getConnLink(msgText.ClientID)
				if err != nil {
					saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprintf("text message, %v", err),
						FuncName:    fn,
					})

					continue
				}

				if err := c.SendWsMessage(1, *msgText.Data); err != nil {
					saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprintf("binary message, %v", err),
						FuncName:    fn,
					})
				}

			case msgBinary := <-cwtResBinary:
				c, err := getConnLink(msgBinary.ClientID)
				if err != nil {
					saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})

					continue
				}

				if err := c.SendWsMessage(2, *msgBinary.Data); err != nil {
					saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})
				}
			}
		}
	}()

	//обработка запросов поступающих в приложение снаружи
	go RouteWssConnect(cwtResText, cwtResBinary, appc, sma, saveMessageApp, cwtReq)

	//запуск телеметрии
	go telemetry.TransmissionTelemetry(cwtResText, appc, sma, saveMessageApp)

	if appc.ForLocalUse {
		go UnixSocketInteraction(cwtReq, appc, sma, saveMessageApp)
	}

	/* запуск приложения в режиме 'СЕРВЕР' */
	if appc.IsServer {
		ServerNetworkInteraction(cwtReq, appc, sma, saveMessageApp)

		return
	}

	/* запуск приложения в режиме 'КЛИЕНТА' */
	ClientNetworkInteraction(cwtReq, appc, sma, conf, saveMessageApp)
}
