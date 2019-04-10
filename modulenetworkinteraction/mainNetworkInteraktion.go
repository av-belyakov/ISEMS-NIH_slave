package modulenetworkinteraction

/*
* Модуль сетевого взаимодействия осуществляет прием или установление
* соединений с клиентами приложения
*
* Версия 0.31, дата релиза 03.04.2019
* */

import (
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gorilla/websocket"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
	"ISEMS-NIH_slave/telemetry"
)

type serverSetting struct {
	IP, Port, Token        string
	StoreMemoryApplication *configure.StoreMemoryApplication
}

type serverWebsocketSetting struct {
	StoreMemoryApplication *configure.StoreMemoryApplication
	Cwt                    chan<- configure.MsgWsTransmission
}

type clientSetting struct {
	ID, IP, Port           string
	StoreMemoryApplication *configure.StoreMemoryApplication
	Cwt                    chan<- configure.MsgWsTransmission
}

func createClientID(str string) string {
	h := md5.New()
	io.WriteString(h, str)

	return hex.EncodeToString(h.Sum(nil))
}

//при разрыве соединения удаляет дескриптор соединения и изменяет статус клиента
func connClose(c *websocket.Conn, sma *configure.StoreMemoryApplication, id, ip string) {
	fmt.Println("CLOSE WSS LINK")

	if c != nil {
		c.Close()
	}

	//удаляем параметры подключения клиента
	sma.DeleteClientSetting(id)
	//удаляем дескриптор соединения
	sma.DelLinkWebsocketConnection(ip)
}

//MainNetworkInteraction модуль сетевого взаимодействия
func MainNetworkInteraction(appc *configure.AppConfig, sma *configure.StoreMemoryApplication) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	//читаем сертификат CA для того что бы клиент доверял сертификату переданному сервером
	rootCA, err := ioutil.ReadFile(appc.LocalServerHTTPS.PathRootCA)
	if err != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	//создаем новый пул доверенных центров серификации и добавляем в него корневой сертификат
	cp := x509.NewCertPool()
	if ok := cp.AppendCertsFromPEM(rootCA); !ok {
		_ = saveMessageApp.LogMessage("error", "root certificate was not added to the pool")
	}

	conf := &tls.Config{
		ServerName: "isems_nih_master",
		RootCAs:    cp,
	}
	conf.BuildNameToCertificate()

	//инициализируем канал для передачи текстовых данных через websocket соединение
	cwtResText := make(chan configure.MsgWsTransmission)
	//инициализируем канал для передачи текстовых данных через websocket соединение
	cwtResBinary := make(chan configure.MsgWsTransmission)
	//инициализируем канал для приема текстовых данных через websocket соединение
	cwtReq := make(chan configure.MsgWsTransmission)

	//обработка ответов поступающих изнутри приложения
	//через канал cwtRes
	go func() {
		getConnLink := func(msg configure.MsgWsTransmission) (*configure.WssConnection, error) {
			s, ok := sma.GetClientSetting(msg.ClientID)
			if !ok {
				return nil, errors.New("the ip address cannot be found by the given client ID " + msg.ClientID)
			}

			//получаем линк websocket соединения
			c, ok := sma.GetLinkWebsocketConnect(s.IP)
			if !ok {
				return nil, errors.New("no connection found at websocket link ip address " + s.IP)
			}

			return c, nil
		}

		for {
			select {
			case msgText := <-cwtResText:

				fmt.Println(msgText.ClientID)

				c, err := getConnLink(msgText)
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

					break
				}
				if err := c.SendWsMessage(1, *msgText.Data); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}

			case msgBinary := <-cwtResBinary:
				c, err := getConnLink(msgBinary)
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

					break
				}
				if err := c.SendWsMessage(2, *msgBinary.Data); err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
				}
			}
		}
	}()

	//обработка запросов поступающих в приложение снаружи
	go RouteWssConnect(cwtResText, cwtResBinary, appc, sma, cwtReq)

	//запуск телеметрии
	go telemetry.TransmissionTelemetry(cwtResText, appc, sma)

	/* запуск приложения в режиме 'СЕРВЕР' */
	if appc.IsServer {
		ServerNetworkInteraction(cwtReq, appc, sma)

		return
	}

	/* запуск приложения в режиме 'КЛИЕНТА' */
	ClientNetworkInteraction(cwtReq, appc, sma, conf)
}
