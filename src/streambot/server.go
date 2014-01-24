package streambot

import (
  "sync"
  "net"
  "net/http"
)

type APIServer struct {
  http.Server
  listener  net.Listener
  waiter    sync.WaitGroup
  // An internal channel to check if the Listener within the Server got closed, as the Server will 
  // react with an error, which needs to be ignored, as it's expected.
  // @see http://goo.gl/kPQKb0
  Closed    chan bool
}
 
func (srv *APIServer) ListenAndServe() error {
  addr := srv.Addr
  if addr == "" {
    addr = ":http"
  }
  var err error
  srv.listener, err = net.Listen("tcp", addr)
  if err != nil {
    return err
  }
  err = srv.Serve(srv.listener)
  return err
}
 
func (srv *APIServer) Serve(l net.Listener) error {
  cur_handler := srv.Handler
  defer func() { 
    srv.Handler = cur_handler
  }()
  new_handler := http.NewServeMux()
  new_handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    srv.waiter.Add(1)
    defer srv.waiter.Done()
    cur_handler.ServeHTTP(w, r)
  })
  srv.Handler = new_handler
  var err error
  serveErr := srv.Server.Serve(l)
  // If there was an error, check if it happened after the Server was killed via closing Listener. 
  // In that case ignore the error.
  if serveErr != nil {
    select {
      case <- srv.Closed:
        // If called Stop() then there will be a value in srv.Closed, so
        // we'll get here and we can exit without showing the error.
      default:
        err = serveErr
    }
  }
  return err
}
 
func (srv *APIServer) Stop() error {
  srv.Closed <- true
  err := srv.listener.Close()
  return err
}
 
func (srv *APIServer) WaitUnfinished() {
  srv.waiter.Wait()
  return
}