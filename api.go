package main 

import ( 
  "code.google.com/p/gorest" 
  "net/http"
)

func main() { 
    service := new(V1)
    gorest.RegisterService(service)
    http.Handle("/",gorest.Handle())     
    http.ListenAndServe(":8080", nil) 
}

//Service Definition 
type V1 struct { 
    gorest.RestService `root:"/v1/" consumes:"application/json" produces:"application/json"` 
    index gorest.EndPoint `method:"GET" path:"/" output:"string"`
}

func(serv V1) Index() (out string) {
    return "Hello world!"
}