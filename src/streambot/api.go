package streambot 

import ( 
  "net/http"
  "net"
  "fmt"
  "time"
  "sync"
  "errors"
  "github.com/laurent22/ripple"
)

type API struct { 
  App       ripple.Application
  GoClose   chan bool
  Server    APIServer
  StatConn  net.Conn
  Closed    chan bool
}

func(api API) Serve(Port int, Route string, ErrorChannel chan error) {
  if Port == 0 {
    ErrorChannel <- errors.New("Cannot spawn server on port 0")
    api.Closed <- true
    return
  }
  // Handle the REST API  
  api.App.SetBaseUrl(Route)
  Address := fmt.Sprintf(":%d", Port)
  var wg sync.WaitGroup
  var ls net.Listener
  api.Server = APIServer {
    http.Server {
      Addr:           Address,
      ReadTimeout:    10 * time.Second,
      WriteTimeout:   10 * time.Second,
    },
    ls,
    wg,
    make(chan bool, 1),
  }
  handler := http.NewServeMux()
  handler.HandleFunc(Route, api.App.ServeHTTP)
  api.Server.Handler = handler
  go func() {
    <- api.GoClose
    api.Server.Stop()
    api.Server.WaitUnfinished()
  }()
  go func() {
    err := api.Server.ListenAndServe()
    if err != nil {
      errMsg := fmt.Sprintf("An error occurred when launching API server: %v", err)
      ErrorChannel <- errors.New(errMsg)
    }
    api.StatConn.Close()
    api.Closed <- true
  }()
}
  
func(api API) Shutdown() {
  api.GoClose <- true
}

func NewAPI(db Database, subsStatsLogfile string) (api *API) {
  api = new(API)
  conn, err := net.Dial("udp", ":8125")
  if err != nil {
    log.Error("Error when instantiate UDP statting connection: %v", err)
  }
  api.StatConn = conn
  app := ripple.NewApplication()
  app.RegisterController("channels", NewChannelController(db, api.StatConn, subsStatsLogfile))
  app.AddRoute(ripple.Route{ Pattern: ":_controller/:id/:_action" })
  app.AddRoute(ripple.Route{ Pattern: ":_controller/:id/" })
  app.AddRoute(ripple.Route{ Pattern: ":_controller" })
  api.App = *app
  api.GoClose = make(chan bool, 1)
  api.Closed = make(chan bool, 1)
	return
}