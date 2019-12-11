package modulenetworkinteraction

/*
* Модуль обеспечивающий websocket соединение в режиме клиента
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
	//инициализируем функцию конструктор для записи лог-файлов
	saveMessageApp := savemessageapp.New()

	fn := "HandlerRequest"

	bodyHTTPResponseError := []byte(`<!DOCTYPE html>
	<html lang="en"
	<head><meta charset="utf-8"><title>Server Nginx</title></head>
	<body><h1>Access denied. For additional information, please contact the webmaster.</h1></body>
	</html>`)

	stringToken := ""
	for headerName := range req.Header {
		if headerName == "Token" {
			stringToken = req.Header[headerName][0]

			continue
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Language", "en")

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

		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("missing or incorrect identification token (сlient ipaddress %v)", req.RemoteAddr),
			FuncName:    fn,
		})

		return
	}

	//если токен валидный добавляем клиента в список и разрешаем ему дальнейшее соединение
	clientID := createClientID(req.RemoteAddr + ":" + ss.Token)
	ss.StoreMemoryApplication.SetClientSetting(clientID, configure.ClientSettings{
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

	fn := "ServerWss"

	remoteIP := strings.Split(req.RemoteAddr, ":")[0]

	clientID, isExistID := sws.StoreMemoryApplication.GetClientIDOnIP(remoteIP)
	if !isExistID {
		w.WriteHeader(401)
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("access for the user with ipaddress %v is prohibited", remoteIP),
			FuncName:    fn,
		})

		return
	}

	//проверяем разрешено ли данному ip соединение с сервером wss
	if !sws.StoreMemoryApplication.GetAccessIsAllowed(remoteIP) {
		w.WriteHeader(401)
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("access for the user with ipaddress %v is prohibited", remoteIP),
			FuncName:    fn,
		})

		return
	}

	if req.Header.Get("Connection") != "Upgrade" {
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
		if c != nil {
			c.Close()
		}

		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}
	defer connClose(c, sws.StoreMemoryApplication, clientID, remoteIP, "server", saveMessageApp)

	fmt.Printf("connection success established, client ID %v, client IP %v\n", clientID, remoteIP)

	//изменяем состояние соединения для данного источника
	if err := sws.StoreMemoryApplication.ChangeSourceConnectionStatus(clientID, true); err != nil {
		_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}

	//добавляем линк соединения по websocket
	sws.StoreMemoryApplication.AddLinkWebsocketConnect(remoteIP, c)

	//обработчик запросов приходящих через websocket
	for {
		if c == nil {
			break
		}

		_, message, err := c.ReadMessage()
		if err != nil {
			_ = saveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
				Description: fmt.Sprint(err),
				FuncName:    fn,
			})

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
	sma *configure.StoreMemoryApplication) {

	log.Printf("START application ISEMS-NIH_slave version %q, the application is running as a \"SERVER\", ip %v, port %v\n", appc.VersionApp, appc.LocalServerHTTPS.IP, appc.LocalServerHTTPS.Port)

	ss := serverSetting{
		IP:                     appc.LocalServerHTTPS.IP,
		Port:                   strconv.Itoa(appc.LocalServerHTTPS.Port),
		Token:                  appc.LocalServerHTTPS.AuthenticationToken,
		StoreMemoryApplication: sma,
	}

	sws := serverWebsocketSetting{
		StoreMemoryApplication: sma,
		Cwt:                    cwtReq,
	}

	http.HandleFunc("/", ss.HandlerRequest)
	http.HandleFunc("/wss", sws.ServerWss)

	if err := http.ListenAndServeTLS(ss.IP+":"+ss.Port, appc.LocalServerHTTPS.PathCertFile, appc.LocalServerHTTPS.PathPrivateKeyFile, nil); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
