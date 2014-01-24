package "acceptance_test"

import(
	"testing"
	"flag"
	"http"
	"streambot/api"
)

var rexsterHost string
var rexsterPort uint16

func init() {
	const (
		defaultRexsterHost = "localhost"
		defaultRexsterPort = 8182
	)
	flag.StringVar(&rexsterHost, "rexster-host", defaultRexsterHost, "Host of the Rexster REST server")
	flag.UintVar(&rexsterPort, "rexster-port", defaultRexsterPort, "Port of the Rexster REST server")
}

type PostChannelResponse struct {
	Id string
}

func TestAPICreateChannelPersistsToTitan(t *testing.T) {

	GRAPH = "foobarbaz"

	db = database.NewGraphDatabase(GRAPH, rexsterHost, rexsterPort)
	var a = api.NewAPI_V1(db)
	a.Start()

	m := PostChannelData{"abc"}
	b, err := json.Marshal(m)
	if err != nil {
		// handle error
	}
	
	resp, err := http.Post("http://localhost:8080/v1/channels", "application/json", bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("%v", string(body))
	
	var postChannelResponse PostChannelResponse
	err = json.Unmarshal(&postChannelResponse, body)
	if err != nil {
		panic(err)
	}
	if postChannelResponse.Id == "" {
		t.Errorf("Unexpected empty Id in API response. Got '"+string(body)+"'")
	}

	url := "http://"+rexsterHost+":"+rexsterPort+"/graph/"+GRAPH+"/vertices/"+postChannelResponse.Id
	resp, err = http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}

	t.Errorf("TBC")
}