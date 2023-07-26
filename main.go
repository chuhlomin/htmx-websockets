// A server that accepts a webhook and sends the payload to all connected
// clients via websocket.
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"text/template"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func (s *server) webhook(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read body: %v", err)
		return
	}

	log.Printf("Received webhook: %s", string(b))

	var buf bytes.Buffer
	err = messageTemplate.Execute(
		&buf,
		struct {
			Raw string
		}{
			Raw: string(b),
		},
	)
	if err != nil {
		log.Printf("Failed to execute template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// wrap the message in a div so we can use htmx to add it to the page
	s.hub.Broadcast <- []byte("<div hx-swap-oob=\"afterbegin:#messages\"><div class=\"message\">" + buf.String() + "</div></div>")

	s.mutex.Lock()
	s.pastMessages = append(s.pastMessages, buf.String())
	if len(s.pastMessages) > 10 {
		s.pastMessages = s.pastMessages[1:]
	}
	log.Printf("Now have %d past messages", len(s.pastMessages))
	s.mutex.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	name := r.Header.Get("User-Agent")
	client, err := NewClient(s.hub, w, r, name)
	if err != nil {
		log.Printf("Failed to create WebSocket client: %v", err)
		return
	}

	s.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(
		w,
		struct {
			WebsocketHost string
			PastMessages  []string
		}{
			WebsocketHost: "ws://" + r.Host + "/events",
			PastMessages:  s.pastMessages,
		},
	)
}

type server struct {
	messageChan  chan string
	pastMessages []string
	mutex        sync.Mutex
	hub          *Hub
}

func main() {
	s := server{
		messageChan:  make(chan string),
		pastMessages: []string{},
		hub:          NewHub(),
	}

	go s.hub.Run()

	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/webhook", s.webhook)
	http.HandleFunc("/events", s.events)
	http.HandleFunc("/sprite.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "sprite.png")
	})
	http.HandleFunc("/", s.home)

	log.Println("Server started at", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script src="https://unpkg.com/htmx.org@1.9.3"></script>
<script src="https://unpkg.com/htmx.org/dist/ext/ws.js"></script>
<style>
body { font-size: 12pt; font-family: sans-serif; background-color: #f0f0ff; }
.message { margin-bottom: 1em; }
.message:nth-child(odd) { background-color: #eee; }
#status {
	font-size: 9pt;
	font-family: SF Mono, monospace;
	color: #8aa487;
}
#status::before {
	content: "";
	display: inline-block;
	width: 25px;
	height: 19px;
	margin-right: 0.2em;
	vertical-align: middle;
	background-image: url(sprite.png);
	background-repeat: no-repeat;
	background-position: left center;
	background-size: 100px 19px;
}
#status[data-status="connected"] { color: #8aa487; }
#status[data-status="error"] { color: #c4796f; }
#status[data-status="connecting"] { color: #c8ad97; }
#status[data-status="disconnected"] { color: #8e8e8e; }

#status[data-status="connected"]::before { background-position: left center; }
#status[data-status="error"]::before { background-position: -25px center; }
#status[data-status="connecting"]::before { background-position: -50px center; }
#status[data-status="disconnected"]::before { background-position: right center; }
</style>
</head>
<body>
<div hx-ext="ws" ws-connect="/events">
	<div id="panel">
		<div id="users"></div>
		<div id="status"></div>
	</div>
	<div id="messages">
		{{ range .PastMessages }}
		<div class="message">{{ . }}</div>
		{{ end }}
	</div>
</div>
<script type="text/javascript" defer>
let status = document.getElementById('status');

// htmx:wsConnecting
// htmx:wsError

document.body.addEventListener('htmx:wsOpen', function(evt) {
	console.log('connected');
	status.innerText = 'Connected';
	status.setAttribute('data-status', 'connected');
});
document.body.addEventListener('htmx:wsClose', function(evt) {
	console.log('disconnected');
	status.innerText = 'Disconnected';
	status.setAttribute('data-status', 'disconnected');
});
</script>
</body>
</html>
`))

var messageTemplate = template.Must(template.New("").Parse(`
	{{ .Raw }}
`))
