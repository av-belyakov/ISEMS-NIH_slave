package modulenetworkinteraction

/*
* Модуль обеспечивающий websocket соединение в режиме клиента
*
* Версия 0.1, дата релиза 02.04.2019
* */

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"

	"github.com/gorilla/websocket"
)

func (cs clientSetting) redirectPolicyFunc(req *http.Request, rl []*http.Request) error {
	fmt.Println("start function REDIRECT")

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	go func() {
		header := http.Header{}
		header.Add("Content-Type", "text/plain;charset=utf-8")
		header.Add("Accept-Language", "en")
		header.Add("User-Agent", "Mozilla/5.0 (ISEMS-NIH_slave)")

		d := websocket.Dialer{
			HandshakeTimeout:  (time.Duration(1) * time.Second),
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		c, res, err := d.Dial("wss://"+cs.IP+":"+cs.Port+"/wss", header)
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			return
		}
		defer connClose(c, cs.InfoSourceList, cs.ID, cs.IP)

		if res.StatusCode == 101 {
			//изменяем статус подключения клиента
			_ = cs.InfoSourceList.ChangeSourceConnectionStatus(cs.ID)

			//добавляем линк соединения
			cs.InfoSourceList.AddLinkWebsocketConnect(cs.IP, c)

			//обработчик запросов приходящих через websocket
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

					break
				}

				cs.Cwt <- configure.MsgWsTransmission{
					ClientID: cs.ID,
					Data:     &message,
				}
			}
		}
	}()

	//отправляем ошибку что бы предотвратить еще один редирект который вызывается после обработки этой функции
	return errors.New("stop redirect")
}

//ClientNetworkInteraction соединение в режиме 'клиент'
func ClientNetworkInteraction(
	cwtReq chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	isl *configure.InformationSourcesList,
	conf *tls.Config) {

	log.Printf("START application ISEMS-NIH_slave version %q, the application is running as a \"CLIENT\"\n", appc.VersionApp)

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	//читаем список доступных к подключению клиентов
	for _, c := range appc.ToConnectServerHTTPS {
		clientID := createClientID(c.IP + ":" + strconv.Itoa(c.Port) + ":" + c.Token)

		isl.AddSourceSettings(clientID, configure.SourceSetting{
			IP:    c.IP,
			Port:  strconv.Itoa(c.Port),
			Token: c.Token,
		})
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
			TLSClientConfig:    conf,
		},
	}

	//цикличные попытки установления соединения в режиме клиент
	ticker := time.NewTicker(time.Duration(appc.TimeReconnect) * time.Second)
	for range ticker.C {
		for id, s := range *isl.GetSourceList() {
			if !s.ConnectionStatus {
				fmt.Printf("connection attempt with client IP: %v, ID %v\n", s.IP, id)

				cs := clientSetting{
					ID:             id,
					IP:             s.IP,
					Port:           s.Port,
					InfoSourceList: isl,
					Cwt:            cwtReq,
				}
				client.CheckRedirect = cs.redirectPolicyFunc

				req, err := http.NewRequest("GET", "https://"+s.IP+":"+s.Port+"/", nil)
				if err != nil {
					_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

					continue
				}

				req.Header.Add("Content-Type", "text/plain;charset=utf-8")
				req.Header.Add("Accept-Language", "en")
				req.Header.Add("User-Agent", "Mozilla/5.0 (ISEMS-NIH_slave)")
				req.Header.Add("Token", s.Token)

				_, err = client.Do(req)
				if err != nil {
					strErr := fmt.Sprint(err)
					if !strings.Contains(strErr, "stop redirect") {
						_ = saveMessageApp.LogMessage("error", strErr)
					}

					continue
				}
			}
		}
	}
}
