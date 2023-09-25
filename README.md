# htmx-websockets

This is a simple example of using WebSockets with [HTMX](https://htmx.org/).  
It uses the [htmx WebSockets extension](https://htmx.org/extensions/web-sockets/)
to connect to a websocket server and echo messages back to the client.

## Server

The server is written in Go and uses the [gorilla/websocket](https://github.com/gorilla/websocket) package to handle WebSockets.

Routes:

* `/` - serves the index page
* `/events` - handles websocket connections
* `/webhook` - handles POST requests and broadcasts the body to all connected clients

Server keeps last 10 messages in memory and broadcasts them to new clients.

## Running the server

```shell
go run .
```

Then open http://localhost:8080 in your browser.

Try opening multiple browsers and see list of connected clients update.

Send a message to all connected clients:

```shell
curl -X POST -d '{"message": "Hello, world!"}' http://localhost:8080/webhook
```
