package main

import (
	"testing"
	"net/http"
	"encoding/json"
	"../src/streambot"
	"bytes"
	"io/ioutil"
	"regexp"
	"fmt"
	"code.google.com/p/go-uuid/uuid"
	"time"
)


//
// TODO
// Refactor that all API instances created tests reside behind the same port. There was an issue 
// with side-effects of database instances between the tests.
//


type PostNewChannelRequest struct {
	Name string `json:"name"`
}

type PostNewChannelResponse struct {
	Id string `json:"id"`
}

type DatabaseMock struct {
	SavedChannel 			streambot.Channel
	SavedSubscription		TestChannelSubscriptionData
	ChannelSubscriptions 	[]streambot.Channel
}

func(db *DatabaseMock) SaveChannel(ch *streambot.Channel) (err error) {
	db.SavedChannel = *ch
	return
}

func(db *DatabaseMock) GetChannelWithUid(uid string) (err error, ch *streambot.Channel) {
	ch = &streambot.Channel{uid, "abc"}
	return
}

type TestChannelSubscriptionData struct {
	FromChannelId 	string
	ToChannelId 	string 
	CreationTime 	int64
}

func(db *DatabaseMock) SaveChannelSubscription(
	fromChannelId string, 
	toChannelId string, 
	creationTime int64,
) (err error) {
	db.SavedSubscription = TestChannelSubscriptionData{fromChannelId, toChannelId, creationTime}
	return
}

func(db *DatabaseMock) GetSubscriptionsForChannelWithUid(uid string) (
	err error, 
	chs []streambot.Channel,
) {
	chs = db.ChannelSubscriptions
	return
}

const UUID_FORMAT = "^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$"

func TestAPIPutChannelSavesChannelInDatabase(t *testing.T) {
	// Define the channel's name
	CHANNEL := "foobarbaz"
	// Instantiate database mock to be used by API server
	db := new(DatabaseMock)
	// Start API HTTP server with database mock
	a := streambot.NewAPI(db)
	errChan := make(chan error)
	a.Serve(8080, "/v1/", errChan)
	go func() {
		err := <- errChan
		t.Fatalf("Unexpected error occurred when starting API: %v", err)
	}()
	t.Logf("Continue")
	// Create a PUT request to store a new Channel
	b, err := json.Marshal(PostNewChannelRequest{CHANNEL})
	if err != nil {
		t.Fatalf("Unexpected error '%v' on marchalling JSON body for Channel creation POST "+
			"request.", err)
	}
	t.Logf("Create Request body: %v", bytes.NewBuffer(b).String())
	url := "http://localhost:8080/v1/channels"
	req, err := http.NewRequest("PUT", url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("Unexpected error when creating PUT request: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	t.Logf("Sending request %v", req)
	cli := &http.Client{}
	res, err := cli.Do(req)
	if err != nil {
		msgFormat := "Unexpected error on executing Channel creation PUT request on URL `%s`: %v"
		t.Fatalf(msgFormat, url, err)
	}
	// Read the request response into a raw buffer
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	t.Logf("Request response body: %v", string(body))
	// Unmarshal buffer to further handle the response
	var response PostNewChannelResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("Unexpected error when unmarshalling JSON response `%s`: %v", string(body), err)
	}
	// Verify response body contains an ID that is of UUID format
	re := regexp.MustCompile(UUID_FORMAT)
    if !re.MatchString(response.Id) {
    	msgFormat := "Expected response body to contain a valid ID of UUID format, given `%s`"
		t.Fatalf(msgFormat, response.Id)
    }
	// Verify saved channel contains expected data from request
	if db.SavedChannel.Name != CHANNEL {
		format := "Saved Channel name `%v` does not match the expected `%v`."
		t.Fatalf(format,  db.SavedChannel.Name, CHANNEL)
	}
	a.Shutdown()
	<- a.Closed
	fmt.Println("Done")
}

type GetChannelResponse struct {
	Id 		string `json:"id"`
	Name 	string `json:"name"`
}

func TestAPIGetChannelRetrievedChannelFromDatabaseSuccess(t *testing.T) {
	// Define the channel's name
	CHANNEL_UID := "foobarbazqux"
	// Instantiate database mock to be used by API server
	db := new(DatabaseMock)
	// Start API HTTP server with database mock
	a := streambot.NewAPI(db)
	errChan := make(chan error)
	a.Serve(8081, "/v1/", errChan)
	go func() {
		err := <- errChan
		t.Fatalf("Unexpected error occurred when starting API: %v", err)
	}()
	url := fmt.Sprintf("http://localhost:8081/v1/channels/%s", CHANNEL_UID)
	res, err := http.Get(url)
	if err != nil {
		msgFormat := "Unexpected error on executing Channel fetch GET request on URL `%s`: %v"
		t.Fatalf(msgFormat, url, err)
	}
	// Read the request response into a raw buffer
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	t.Logf("Request response body: %v", string(body))
	// Unmarshal buffer to further handle the response
	var response GetChannelResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("Unexpected error when unmarshalling JSON response `%s`: %v", string(body), err)
	}
	err, ch := db.GetChannelWithUid(CHANNEL_UID)
	if response.Name != ch.Name {
		errMsgFormat := "Expected API response to contain channel's name `%s`, given `%s`"
		t.Fatalf(errMsgFormat, ch.Name, response.Name)
	}
	if response.Id != ch.Id {
		errMsgFormat := "Expected API response to contain channel's ID `%s`, given `%s`"
		t.Fatalf(errMsgFormat, ch.Id, response.Id)
	}
	a.Shutdown()
	<- a.Closed
	fmt.Println("Done")
}

type PostChannelSubscriptionRequest struct {
	ChannelId 	string 	`json:"channel_id"`
	Time 		int64 	`json:"created_at"`
}

func TestAPIPostChannelSubscriptionInDatabaseSuccess(t *testing.T) {
	// Define ID of channel subscribing
	FROM_CHANNEL_ID := uuid.New()
	// Define ID of channel to get subscribed
	TO_CHANNEL_ID := uuid.New()
	// Define timestamp of the subscription was initiated
	SUBSCRIPTION_TIME := time.Now().Unix()
	// Instantiate database mock to be used by API server
	db := new(DatabaseMock)
	// Start API HTTP server with database mock
	a := streambot.NewAPI(db)
	errChan := make(chan error)
	a.Serve(8082, "/v1/", errChan)
	go func() {
		t.Fatalf("Unexpected error occurred when starting API: %v", <- errChan)
	}()
	url := fmt.Sprintf("http://localhost:8082/v1/channels/%s/subscriptions", FROM_CHANNEL_ID)
	t.Logf("Channel subscription POST URL: %s", url)
	// Create a POST request to store a new subscription
	b, err := json.Marshal(PostChannelSubscriptionRequest{TO_CHANNEL_ID, SUBSCRIPTION_TIME})
	if err != nil {
		t.Fatalf("Unexpected error '%v' on marchalling JSON body for Channel subscription POST "+
			"request.", err)
	}
	t.Logf("Create Request body: %v", bytes.NewBuffer(b).String())
	res, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		msgFormat := "Unexpected error on Channel subscription request on URL `%s`: %v"
		t.Fatalf(msgFormat, url, err)
	}
	if res.StatusCode != 200 {
		msgFormat := "Unexpected status code on Channel subscription response on URL `%d`: 200"
		t.Fatalf(msgFormat, res.StatusCode)	
	}
    t.Logf("Database %v", db)
	// Verify saved channel subscription contains expected data from request
	if db.SavedSubscription.CreationTime != SUBSCRIPTION_TIME {
		format := "Saved Channel subscription time `%d` does not match the expected `%d`."
		t.Fatalf(format, db.SavedSubscription.CreationTime, SUBSCRIPTION_TIME)
	}
	if db.SavedSubscription.ToChannelId != TO_CHANNEL_ID {
		format := "Saved Channel subscription target channel `%s` does not match the expected `%s`."
		t.Fatalf(format,  db.SavedSubscription.ToChannelId, TO_CHANNEL_ID)
	}
	if db.SavedSubscription.FromChannelId != FROM_CHANNEL_ID {
		format := "Saved Channel subscription source channel `%s` does not match the expected `%s`."
		t.Fatalf(format,  db.SavedSubscription.FromChannelId, FROM_CHANNEL_ID)
	}
	a.Shutdown()
	<- a.Closed
	fmt.Println("Done")
}

func TestAPIGetChannelSubscriptionsFromDatabaseSucess(t *testing.T) {
	// Define the subscribing channel's name
	CHANNEL_UID := "foobarbazqux"
	// Instantiate database mock to be used by API server
	db := new(DatabaseMock)
	db.ChannelSubscriptions = []streambot.Channel{streambot.Channel{Id: uuid.New(), Name: uuid.New()}}
	// Start API HTTP server with database mock
	a := streambot.NewAPI(db)
	errChan := make(chan error)
	a.Serve(8083, "/v1/", errChan)
	go func() {
		err := <- errChan
		t.Fatalf("Unexpected error occurred when starting API: %v", err)
	}()
	url := fmt.Sprintf("http://localhost:8083/v1/channels/%s/subscriptions", CHANNEL_UID)
	res, err := http.Get(url)
	if err != nil {
		msgFormat := "Unexpected error on executing Channel subscriptions fetch GET request" +
		" on URL `%s`: %v"
		t.Fatalf(msgFormat, url, err)
	}
	// Read the request response into a raw buffer
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	t.Logf("Request response body: %v", string(body))
	// Unmarshal buffer to further handle the response
	var response []streambot.Channel
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("Unexpected error when unmarshalling JSON response `%s`: %v", string(body), err)
	}
	err, chs := db.GetSubscriptionsForChannelWithUid(CHANNEL_UID)
	if len(chs) != len(db.ChannelSubscriptions) {
		errMsgFormat := "Expected API to return exact same amount of subscribed channels as " +
		"returned by database: %d, given %d"
		t.Fatalf(errMsgFormat, len(chs), len(db.ChannelSubscriptions))
	}
	for i := range chs {
		channel := response[i]
		if chs[i].Name != channel.Name {
			errMsgFormat := "Expected API response to contain channel's name `%s`, given `%s`"
			t.Fatalf(errMsgFormat, chs[i].Name, channel.Name)
		}
		if chs[i].Id != channel.Id {
			errMsgFormat := "Expected API response to contain channel's ID `%s`, given `%s`"
			t.Fatalf(errMsgFormat, chs[i].Id, channel.Id)
		}
	}
	a.Shutdown()
	<- a.Closed
	fmt.Println("Done")

}