package modulenetworkinteraction

/*
* Модуль сетевого взаимодействия осуществляет прием или установление
* соединений с клиентами приложения
*
* Версия 0.3, дата релиза 02.04.2019
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

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

type serverSetting struct {
	IP, Port, Token string
	InfoSourceList  *configure.InformationSourcesList
}

type serverWebsocketSetting struct {
	InfoSourceList *configure.InformationSourcesList
	Cwt            chan<- configure.MsgWsTransmission
}

type clientSetting struct {
	ID, IP, Port   string
	InfoSourceList *configure.InformationSourcesList
	Cwt            chan<- configure.MsgWsTransmission
}

func createClientID(str string) string {
	h := md5.New()
	io.WriteString(h, str)

	return hex.EncodeToString(h.Sum(nil))
}

//HandlerRequest обработчик HTTPS запросов
/*func (ss *serverSetting) HandlerRequest(w http.ResponseWriter, req *http.Request) {

	fmt.Println("START function 'HandlerRequest'...")

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	bodyHTTPResponseError := []byte(`<!DOCTYPE html>
	<html lang="en"
	<head><meta charset="utf-8"><title>Server Nginx</title></head>
	<body><h1>Access denied. For additional information, please contact the webmaster.</h1></body>
	</html>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", "en")

	stringToken := ""
	for headerName := range req.Header {
		if headerName == "Token" {
			stringToken = req.Header[headerName][0]

			continue
		}
	}

	if req.Method != "GET" {
		http.Error(w, "Method not allowed", 405)

		return
	}

	userSettings := strings.Split(req.RemoteAddr, ":")
	remoteIP, remotePort := userSettings[0], userSettings[1]

	if (len(stringToken) == 0) || (stringToken != ss.Token) {
		w.Header().Set("Content-Length", strconv.Itoa(utf8.RuneCount(bodyHTTPResponseError)))

		w.WriteHeader(400)
		w.Write(bodyHTTPResponseError)

		_ = saveMessageApp.LogMessage("error", "missing or incorrect identification token (сlient ipaddress "+req.RemoteAddr+")")

		return
	}

	//если токен валидный добавляем клиента в список и разрешаем ему дальнейшее соединение
	clientID := createClientID(req.RemoteAddr + ":" + ss.Token)
	ss.InfoSourceList.AddSourceSettings(clientID, configure.SourceSetting{
		IP:              remoteIP,
		Port:            remotePort,
		AccessIsAllowed: true,
	})

	http.Redirect(w, req, "https://"+ss.IP+":"+ss.Port+"/wss", 301)
}

//ServerWss webSocket запросов
func (sws serverWebsocketSetting) ServerWss(w http.ResponseWriter, req *http.Request) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	remoteIP := strings.Split(req.RemoteAddr, ":")[0]

	clientID, idIsExist := sws.InfoSourceList.GetSourceIDOnIP(remoteIP)
	if !idIsExist {
		w.WriteHeader(401)
		_ = saveMessageApp.LogMessage("error", "access for the user with ipaddress "+remoteIP+" is prohibited")
		return
	}

	//проверяем разрешено ли данному ip соединение с сервером wss
	if !sws.InfoSourceList.GetAccessIsAllowed(remoteIP) {
		w.WriteHeader(401)
		_ = saveMessageApp.LogMessage("error", "access for the user with ipaddress "+remoteIP+" is prohibited")
		return
	}

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
		//ReadBufferSize:    1024,
		//WriteBufferSize:   100000000,
		HandshakeTimeout: (time.Duration(1) * time.Second),
	}

	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		c.Close()

		_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))
	}

	defer func() {
		//закрытие канала связи с источником
		c.Close()

		if id, ok := sws.InfoSourceList.GetSourceIDOnIP(remoteIP); ok {
			//изменяем состояние соединения для данного источника
			_ = sws.InfoSourceList.ChangeSourceConnectionStatus(id)
		}

		//удаляем линк соединения
		sws.InfoSourceList.DelLinkWebsocketConnection(remoteIP)

		_ = saveMessageApp.LogMessage("info", "disconnect for IP address "+remoteIP)

		//при разрыве соединения отправляем модулю routing сообщение об изменении статуса источников
		//		sws.MsgChangeSourceConnection <- [2]string{remoteIP, "disconnect"}

		_ = saveMessageApp.LogMessage("info", "websocket disconnect whis ip "+remoteIP)
	}()

	//изменяем состояние соединения для данного источника
	_ = sws.InfoSourceList.ChangeSourceConnectionStatus(clientID)

	//добавляем линк соединения по websocket
	sws.InfoSourceList.AddLinkWebsocketConnect(remoteIP, c)

	//отправляем модулю routing сообщение об изменении статуса источника
	//	sws.MsgChangeSourceConnection <- [2]string{remoteIP, "connect"}

	if e := recover(); e != nil {
		_ = saveMessageApp.LogMessage("error", fmt.Sprint(e))
	}
}

/*func (cs clientSetting) redirectPolicyFunc(req *http.Request, rl []*http.Request) error {
	fmt.Println("start function REDIRECT")

	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	//при разрыве соединения удаляет дескриптор соединения и изменяет статус клиента
	connClose := func(c *websocket.Conn, cs clientSetting, id, ip string) {
		fmt.Println("CREATE WSS LINK")

		c.Close()

		//изменяем статус подключения клиента
		_ = cs.InfoSourceList.ChangeSourceConnectionStatus(id)
		//удаляем дескриптор соединения
		cs.InfoSourceList.DelLinkWebsocketConnection(ip)
	}

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
		defer connClose(c, cs, cs.ID, cs.IP)

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
}*/

//MainNetworkInteraction модуль сетевого взаимодействия
func MainNetworkInteraction(appc *configure.AppConfig) {
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	//инициализируем хранилище подключений клиентов
	isl := configure.NewRepositoryISL()

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
			s, ok := isl.GetSourceSetting(msg.ClientID)
			if !ok {
				return nil, errors.New("the ip address cannot be found by the given client ID " + msg.ClientID)
			}

			//получаем линк websocket соединения
			c, ok := isl.GetLinkWebsocketConnect(s.IP)
			if !ok {
				return nil, errors.New("no connection found at websocket link ip address " + s.IP)
			}

			return c, nil
		}

		for {
			select {
			case msgText := <-cwtResText:
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
	go RouteWssConnect(cwtResText, cwtResBinary, appc, cwtReq)

	/* запуск приложения в режиме 'СЕРВЕР' */
	if appc.IsServer {
		ServerNetworkInteraction(cwtReq, appc, isl)

		/*		log.Printf("START application ISEMS-NIH_slave version %q, the application is running as a \"SERVER\"\n", appc.VersionApp)

				ss := serverSetting{
					IP:             appc.LocalServerHTTPS.IP,
					Port:           strconv.Itoa(appc.LocalServerHTTPS.Port),
					Token:          appc.LocalServerHTTPS.AuthenticationToken,
					InfoSourceList: isl,
				}

				sws := serverWebsocketSetting{
					InfoSourceList: isl,
					Cwt:            cwtReq,
				}

				http.HandleFunc("/", ss.HandlerRequest)
				http.HandleFunc("/wss", sws.ServerWss)

				err = http.ListenAndServeTLS(ss.IP+":"+ss.Port, appc.LocalServerHTTPS.PathCertFile, appc.LocalServerHTTPS.PathPrivateKeyFile, nil)

				log.Println(err)
				os.Exit(1)*/
	}

	/* запуск приложения в режиме 'КЛИЕНТА' */
	ClientNetworkInteraction(cwtReq, appc, isl, conf)

	/*log.Printf("START application ISEMS-NIH_slave version %q, the application is running as a \"CLIENT\"\n", appc.VersionApp)

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
	}*/
}
