package modulenetworkinteraction

/*
* Модуль обеспечивающий websocket соединение в режиме клиента
*
* Версия 0.1, дата релиза 02.04.2019
* */
import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"

	"ISEMS-NIH_slave/configure"
	"ISEMS-NIH_slave/savemessageapp"
)

//HandlerRequest обработчик HTTPS запросов
func (ss *serverSetting) HandlerRequest(w http.ResponseWriter, req *http.Request) {

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

		//изменяем состояние соединения для данного источника
		_ = sws.InfoSourceList.ChangeSourceConnectionStatus(clientID)

		//удаляем линк соединения
		sws.InfoSourceList.DelLinkWebsocketConnection(remoteIP)

		_ = saveMessageApp.LogMessage("info", "websocket disconnect whis ip "+remoteIP)
	}()

	//изменяем состояние соединения для данного источника
	_ = sws.InfoSourceList.ChangeSourceConnectionStatus(clientID)

	//добавляем линк соединения по websocket
	sws.InfoSourceList.AddLinkWebsocketConnect(remoteIP, c)

	//обработчик запросов приходящих через websocket
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			_ = saveMessageApp.LogMessage("error", fmt.Sprint(err))

			break
		}

		sws.Cwt <- configure.MsgWsTransmission{
			ClientID: clientID,
			Data:     &message,
		}
	}
}

//ServerNetworkInteraction соединение в режиме 'сервер'
func ServerNetworkInteraction(
	cwtReq chan<- configure.MsgWsTransmission,
	appc *configure.AppConfig,
	isl *configure.InformationSourcesList) {

	fmt.Println("START function 'ServerNetworkInteraction'...")

	log.Printf("START application ISEMS-NIH_slave version %q, the application is running as a \"SERVER\"\n", appc.VersionApp)

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

	err := http.ListenAndServeTLS(ss.IP+":"+ss.Port, appc.LocalServerHTTPS.PathCertFile, appc.LocalServerHTTPS.PathPrivateKeyFile, nil)

	log.Println(err)
	os.Exit(1)
}
