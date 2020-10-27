package modulenetworkinteraction

/*
* Модуль обеспечивающий websocket соединение в режиме клиента
* */

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"

	"github.com/gorilla/websocket"
)

func (cs clientSetting) redirectPolicyFunc(req *http.Request, rl []*http.Request) error {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	fn := "redirectPolicyFunc"

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
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

			return
		}
		defer connClose(c, cs.StoreMemoryApplication, cs.ID, cs.IP, "client", saveMessageApp)

		if res.StatusCode == 101 {
			//изменяем статус подключения клиента
			if err := cs.StoreMemoryApplication.ChangeSourceConnectionStatus(cs.ID, true); err != nil {
				_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
					Description: fmt.Sprint(err),
					FuncName:    fn,
				})
			}

			fmt.Printf("-= connection with source IP %v success established =-\n", cs.IP)

			//добавляем линк соединения
			cs.StoreMemoryApplication.AddLinkWebsocketConnect(cs.IP, c)
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				TypeMessage: "info",
				Description: fmt.Sprintf("connection with source IP %v success established", cs.IP),
				FuncName:    fn,
			})

			//fmt.Println("проверяем список всех соединений по Websocket")
			//fmt.Println(cs.StoreMemoryApplication.GetClientsListConnection())

			//обработчик запросов приходящих через websocket
			for {
				_, message, err := c.ReadMessage()
				if err != nil {
					_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})

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
	sma *configure.StoreMemoryApplication,
	conf *tls.Config) {

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	fn := "clientNetworkInteraction"

	//читаем список доступных к подключению клиентов
	for _, c := range appc.ToConnectServerHTTPS {
		clientID := createClientID(c.IP + ":" + strconv.Itoa(c.Port) + ":" + c.Token)

		sma.SetClientSetting(clientID, configure.ClientSettings{
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
		for id, s := range sma.GetAllClientSettings() {
			if !s.ConnectionStatus {
				cs := clientSetting{
					ID:                     id,
					IP:                     s.IP,
					Port:                   s.Port,
					StoreMemoryApplication: sma,
					Cwt:                    cwtReq,
				}
				client.CheckRedirect = cs.redirectPolicyFunc

				req, err := http.NewRequest("GET", "https://"+s.IP+":"+s.Port+"/", nil)
				if err != nil {
					_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
						Description: fmt.Sprint(err),
						FuncName:    fn,
					})

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
						_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
							Description: strErr,
							FuncName:    fn,
						})
					}

					continue
				}
			}
		}
	}
}
