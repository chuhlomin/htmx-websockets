// A server that accepts a webhook and sends the payload to all connected
// clients via websocket.
package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func (s Server) webhook(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		return
	}

	log.Printf("Received webhook: %s", string(b))

	s.hub.Broadcast <- b

	w.WriteHeader(http.StatusOK)
}

func (s Server) events(w http.ResponseWriter, r *http.Request) {
	client, err := NewClient(s.hub, w, r)
	if err != nil {
		log.Printf("Failed to create WebSocket client: %v", err)
		return
	}

	s.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

func (s Server) home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(
		w,
		struct {
			WebsocketHost string
			PastMessages  []string
		}{
			WebsocketHost: "wss://" + r.Host + "/events",
			PastMessages:  s.pastMessages,
		},
	)
}

type Server struct {
	messageChan  chan string
	pastMessages []string
	hub          *Hub
}

func main() {
	s := Server{
		messageChan:  make(chan string),
		pastMessages: make([]string, 10),
		hub:          NewHub(),
	}

	go s.hub.Run()

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/webhook", s.webhook)
	http.HandleFunc("/events", s.events)
	http.HandleFunc("/", s.home)

	log.Println("Server started at", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {

    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
        output.scroll(0, output.scrollHeight);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.WebsocketHost}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 70vh;overflow-y: scroll;">
{{ range .PastMessages }}
<div>{{ . }}</div>
{{ end }}
</div>
</td></tr></table>
</body>
</html>
`))
