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

		ss.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("missing or incorrect identification token (сlient ipaddress %v)", req.RemoteAddr),
			FuncName:    fn,
		})

		return
	}

	//если токен валидный добавляем клиента в список и разрешаем ему дальнейшее соединение
	//ss.StoreMemoryApplication.SetClientSetting(createClientID(req.RemoteAddr+":"+ss.Token), configure.ClientSettings{
	ss.StoreMemoryApplication.SetClientSetting(createClientID(remoteIP+":"+ss.Token), configure.ClientSettings{
		IP:              remoteIP,
		Port:            remotePort,
		AccessIsAllowed: true,
	})

	http.Redirect(w, req, "https://"+ss.IP+":"+ss.Port+"/wss", 301)
}

//ServerWss webSocket запросов
func (sws serverWebsocketSetting) ServerWss(w http.ResponseWriter, req *http.Request) {
	fn := "ServerWss"

	remoteIP := strings.Split(req.RemoteAddr, ":")[0]

	clientID, err := sws.StoreMemoryApplication.GetClientIDOnIP(remoteIP)
	if err != nil {
		w.WriteHeader(401)
		sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("access for the user with ip address %v is prohibited", remoteIP),
			FuncName:    fn,
		})

		return
	}

	//проверяем разрешено ли данному ip соединение с сервером wss
	accessIsAllowed, err := sws.StoreMemoryApplication.GetAccessIsAllowed(remoteIP)
	if err != nil {
		w.WriteHeader(401)
		sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})

		return
	}

	if !accessIsAllowed {
		w.WriteHeader(401)
		sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("access for the user with ip address %v is prohibited, the ip address %v is not in the list of allowed addresses", remoteIP, remoteIP),
			FuncName:    fn,
		})

		return
	}

	if req.Header.Get("Connection") != "Upgrade" {
		sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprintf("access for the user with ip address %v is prohibited, the header 'Connection' is not equal to 'Upgrade'", remoteIP),
			FuncName:    fn,
		})

		return
	}

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: false,
		//ReadBufferSize:    1024,
		//WriteBufferSize:   100000000,
		HandshakeTimeout: (time.Duration(3) * time.Second),
	}

	c, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		if c != nil {
			c.Close()
		}

		sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
			Description: fmt.Sprint(err),
			FuncName:    fn,
		})
	}
	defer connClose(c, sws.StoreMemoryApplication, clientID, remoteIP, "server", sws.SaveMessageApp)

	sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
		TypeMessage: "info",
		Description: fmt.Sprintf("connection success established, client ID %v, client IP %v\n", clientID, remoteIP),
		FuncName:    fn,
	})

	//изменяем состояние соединения для данного источника
	if err := sws.StoreMemoryApplication.ChangeSourceConnectionStatus(clientID, true); err != nil {
		sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
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

			fmt.Printf("func11 '%v', err: '%v'\n", fn, err)

			sws.SaveMessageApp.LogMessage(savemessageapp.TypeLogMessage{
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
	sma *configure.StoreMemoryApplication,
	saveMessageApp *savemessageapp.PathDirLocationLogFiles) {

	log.Printf("START application ISEMS-NIH_slave version %q, the application is running as a \"SERVER\", ip %v, port %v\n", appc.VersionApp, appc.LocalServerHTTPS.IP, appc.LocalServerHTTPS.Port)

	ss := serverSetting{
		IP:                     appc.LocalServerHTTPS.IP,
		Port:                   strconv.Itoa(appc.LocalServerHTTPS.Port),
		Token:                  appc.LocalServerHTTPS.AuthenticationToken,
		StoreMemoryApplication: sma,
		SaveMessageApp:         saveMessageApp,
	}

	sws := serverWebsocketSetting{
		StoreMemoryApplication: sma,
		SaveMessageApp:         saveMessageApp,
		Cwt:                    cwtReq,
	}

	http.HandleFunc("/", ss.HandlerRequest)
	http.HandleFunc("/wss", sws.ServerWss)

	if err := http.ListenAndServeTLS(ss.IP+":"+ss.Port, appc.LocalServerHTTPS.PathCertFile, appc.LocalServerHTTPS.PathPrivateKeyFile, nil); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
