package main

import (
	"testing"
	"../src/streambot"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"regexp"
	"fmt"
	"math/rand"
	"code.google.com/p/go-uuid/uuid"
	"time"
)

type NewChannelRequestBody struct {
	Id 		string `json:"_id"`
	Name 	string `json:"name"`
	Uid		string `json:"uid"`
}

const UUID_FORMAT = "^[a-z0-9]{8}-[a-z0-9]{4}-[1-5][a-z0-9]{3}-[a-z0-9]{4}-[a-z0-9]{12}$"

func MockRexsterServerAndInstantiateGraphDatabase(
	t *testing.T, 
	graph string, 
	handler func(w http.ResponseWriter, r *http.Request),
) (err error, r *httptest.Server, db *streambot.GraphDatabase) {
	// Set up a mock server to handle vertex creation request
	r = httptest.NewServer(http.HandlerFunc(handler))
	t.Logf("Server spawned on %s", r.URL)
	// Determine server host and port to initialize graph database
	serverUrlParts := strings.Split(strings.Split(r.URL, "http://")[1], ":")
	host := serverUrlParts[0]
	port, err := strconv.Atoi(serverUrlParts[1])
	if err == nil {
		// Initialize graph database to be verified
		db = streambot.NewGraphDatabase(graph, host, uint16(port))
	}
	return 
}

func RequestBody(t *testing.T, r *http.Request) (body NewChannelRequestBody) {
	// Read binary body content
	rawData, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// Unmarshal binary body, assuming JSON format
	err = json.Unmarshal(rawData, &body)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	return
}

func TestSaveNewChannelInGraph(t *testing.T) {

	GRAPH 		:= "foobarbaz"
	CHANNEL 	:= "foobarbaz"
	ch 			:= streambot.NewChannel(CHANNEL)

	// Keep track on the server side being called up during test
	serverCalled := false

	// Set up a mock server to handle vertex creation request
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Create a target container to capture request body for investigation
		body := RequestBody(t, r)
		t.Logf("Received request on %s with %v", r.URL, body)
		// Verify request body contains expected Channel name
		if body.Name != CHANNEL {
			msgFormat := "Expected request body to contain expected Channel name `%s`, given `%s`"
			t.Fatal(msgFormat, body.Name)
		}
		// Verify request body contains an ID that is of UUID format
		re := regexp.MustCompile(UUID_FORMAT)
        if !re.MatchString(body.Uid) {
        	msgFormat := "Expected request body to contain a valid ID of UUID format, given `%s`"
			t.Fatalf(msgFormat, body.Uid)
        }
        // Verify request method is POST
        if r.Method != "POST" {
        	msgFormat := "Expected request method to be `POST`, given `%s`"
			t.Fatalf(msgFormat, r.Method)
        }
        // Verify URL is of expected shape
        expectedURL := fmt.Sprintf("/graphs/%s/vertices/", GRAPH)
        if r.URL.String() != expectedURL {
        	msgFormat := "Expected request URL to be `%s`, given `%s`"
			t.Fatalf(msgFormat, expectedURL, r.URL.String())
        }
		t.Logf("Received request on %s with %v", r.URL, body)
		// Return an empty string to gain a 200
		resFormat := "{\"version\":\"2.4.0\",\"results\":[{\"uid\":\"%s\",\"name\":\"%s\"," +
		"\"_id\":%s,\"_type\":\"vertex\"}],\"totalSize\":1,\"queryTime\":108.974156}"
		res := fmt.Sprintf(resFormat, body.Uid, CHANNEL, fmt.Sprintf("%v", rand.Intn(10000000000)))
		t.Logf("Server responds with result body `%s`", res)
		fmt.Fprintln(w, res)

		// Switch tracker flag for server call
		serverCalled = true
	}
	err, r, db := MockRexsterServerAndInstantiateGraphDatabase(t, GRAPH, handler)
	defer r.Close()
	if err != nil {
		t.Fatalf("Unexpected error in MockRexsterServerAndInstantiateGraphDatabase: %v", err)
	}
	// Save a channel in the graph
	err = db.SaveChannel(ch)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// Update Channel properties and save again
	CHANNEL = "Foo"
	ch.Name = CHANNEL
	// Save a channel in the graph
	err = db.SaveChannel(ch)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// Verify server was called as expected
	if !serverCalled {
		t.Fatalf("Expected to have posted Channel vertex data to graph database server")
	}
}

func TestGetChannelInGraph(t *testing.T) {

	GRAPH 		:= "foobarbaz"
	CHANNEL 	:= uuid.New()
	var channelUid string

	// Keep track on the server side being called up during test
	serverCalled := false

	// Set up a mock server to handle retrieval request
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Received request on %s", r.URL)
		// Verify URL is of expected shape
		expectedURL := fmt.Sprintf("/graphs/%s/vertices?key=uid&value=%s", GRAPH, channelUid)
		if r.URL.String() == expectedURL {
				// Verify request method is GET
	        if r.Method != "GET" {
	        	msgFormat := "Expected request method to be `GET`, given `%s`"
				t.Fatalf(msgFormat, r.Method)
	        }
	        // Return an empty string to gain a 200
			resFormat := "{\"version\":\"2.4.0\",\"results\":[{\"uid\":\"%s\",\"name\":" +
			"\"%s\",\"_id\":%d,\"_type\":\"vertex\"}],\"totalSize\":1,\"queryTime\"" +
			":108.974156}"
			res := fmt.Sprintf(resFormat, channelUid, CHANNEL, rand.Intn(10000000000))
			fmt.Fprintln(w, res)
		} else {
			http.NotFound(w, r)
		}
		// Switch tracker flag for server call
		serverCalled = true
	}
	err, r, db := MockRexsterServerAndInstantiateGraphDatabase(t, GRAPH, handler)
	defer r.Close()
	if err != nil {
		t.Fatalf("Unexpected error in MockRexsterServerAndInstantiateGraphDatabase: %v", err)
	}
	// Create a new channel
	ch := streambot.NewChannel(CHANNEL)
	// Assume the channel was saved once
	// Capture the channel's ID
	channelUid = ch.Id
	// Now try to get the channel from database
	err, retrievedChannel := db.GetChannelWithUid(channelUid)
	if err != nil {
		t.Fatalf("Unexpected error when retrieving Channel: %v", err)
	}
	if retrievedChannel == nil {
		t.Fatalf("Expected a Channel to be retrieved")
	}
	if retrievedChannel.Name != CHANNEL {
		msgFormat := "Expected retrieved Channel to have name `%s`, given `%s`"
		t.Fatalf(msgFormat, CHANNEL, retrievedChannel.Name)
	}
	// Verify server was called as expected
	if !serverCalled {
		t.Fatalf("Expected to have posted Channel vertex data to graph database server")
	}
}

func TestSaveNewChannelSubscriptionInGraph(t *testing.T) {

	GRAPH 				:= "foobarbaz"
	FROM_CHANNEL_UID 	:= uuid.New()
	FROM_CHANNEL_ID		:= rand.Intn(10000000000)
	TO_CHANNEL_UID 		:= uuid.New()
	TO_CHANNEL_ID		:= rand.Intn(10000000000)
	TIME 				:= time.Now().Unix()
	SUBSCRIPTION_ID 	:= uuid.New()

	// Keep track on the server side being called up during test
	serverCalled := false

	// Set up a mock server to handle subscription edge creation request
	handler := func(w http.ResponseWriter, r *http.Request) {
        // Verify request method is GET
        if r.Method != "GET" {
        	msgFormat := "Expected request method to be `GET`, given `%s`"
			t.Fatalf(msgFormat, r.Method)
        }
        // Verify URL is of expected shape
        expectedURL := fmt.Sprintf("/graphs/%s/tp/gremlin", GRAPH)
        script := fmt.Sprintf(
        	"subs=g.V('uid', '%s')" +
				".out('subscribe').has('id', g.V('uid', '%s').next().id);" +
				"if(!subs.hasNext()){" +
				"e=g.addEdge(g.V('uid','%s').next(),g.V('uid','%s').next()," +
				"'subscribe',[time:%d]);g.commit();e" +
				"}else{g.V('uid', '%s').outE('subscribe')}", 
			FROM_CHANNEL_UID, TO_CHANNEL_UID, FROM_CHANNEL_UID, 
			TO_CHANNEL_UID, TIME, FROM_CHANNEL_UID,)
		q := url.Values{"script": []string{script}}
		expectedURL = fmt.Sprintf("%s?%s", expectedURL, q.Encode())
        if r.URL.String() != expectedURL {
        	msgFormat := "Expected request URL to be `%s`, given `%s`"
			t.Fatalf(msgFormat, expectedURL, r.URL.String())
        }
		t.Logf("Received request on %s", r.URL)
		// Return an empty string to gain a 200
		resFormat := "{\"results\":[{\"time\":%d,\"_id\":\"%s\",\"_type\":\"edge\"," +
		"\"_outV\":%d,\"_inV\":%d,\"_label\":\"subscribe\"}],\"success\":true,\"version\":" +
		"\"2.4.0\",\"queryTime\":23.049033}"
		res := fmt.Sprintf(resFormat, TIME, SUBSCRIPTION_ID, FROM_CHANNEL_ID, TO_CHANNEL_ID)
		fmt.Fprintln(w, res)
		// Switch tracker flag for server call
		serverCalled = true
	}
	err, r, db := MockRexsterServerAndInstantiateGraphDatabase(t, GRAPH, handler)
	defer r.Close()
	if err != nil {
		t.Fatalf("Unexpected error in MockRexsterServerAndInstantiateGraphDatabase: %v", err)
	}
	// Save channel subscription in the graph database
	err = db.SaveChannelSubscription(FROM_CHANNEL_UID, TO_CHANNEL_UID, TIME)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// Verify server was called as expected
	if !serverCalled {
		t.Fatalf("Expected to have posted Channel subscription data to graph database server")
	}
}

func TestGetChannelSubscriptions (t* testing.T) {
	GRAPH 					:= "foobarbar"
	CHANNEL_ID 				:= uuid.New()
	SUBSCRIBED_CHANNEL_ID 	:= uuid.New()
	SUBSCRIBED_CHANNEL_NAME := "foobar"

	// Keep track on the server side being called up during test
	serverCalled := false

	// Set up a mock server to handle subscription edge creation request
	handler := func(w http.ResponseWriter, r *http.Request) {
        // Verify request method is GET
        if r.Method != "GET" {
        	msgFormat := "Expected request method to be `GET`, given `%s`"
			t.Fatalf(msgFormat, r.Method)
        }
        // Verify URL is of expected shape
        expectedURL := fmt.Sprintf("/graphs/%s/tp/gremlin", GRAPH)
        script := fmt.Sprintf("g.V(\"uid\",%s).out.loop(1){it.loops < 100}{true}.dedup", CHANNEL_ID)
		q := url.Values{"script": []string{script}}
		expectedURL = fmt.Sprintf("%s?%s", expectedURL, q.Encode())
        if r.URL.String() != expectedURL {
        	msgFormat := "Expected request URL to be `%s`, given `%s`"
			t.Fatalf(msgFormat, expectedURL, r.URL.String())
        }
		t.Logf("Received request on %s", r.URL)
		// Return an empty string to gain a 200
		resFormat := "{\"results\":[{\"name\":\"%s\",\"uid\":\"%s\",\"_id\":8,\"_type\":" +
		"\"vertex\"}],\"success\":true,\"version\":\"2.4.0\",\"queryTime\":29.266926}"
		res := fmt.Sprintf(resFormat, SUBSCRIBED_CHANNEL_NAME, SUBSCRIBED_CHANNEL_ID)
		fmt.Fprintln(w, res)
		// Switch tracker flag for server call
		serverCalled = true
	}
	err, r, db := MockRexsterServerAndInstantiateGraphDatabase(t, GRAPH, handler)
	defer r.Close()
	if err != nil {
		t.Fatalf("Unexpected error in MockRexsterServerAndInstantiateGraphDatabase: %v", err)
	}
	// Save channel subscription in the graph database
	err, channels := db.GetSubscriptionsForChannelWithUid(CHANNEL_ID)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	fmt.Printf("Channels in test received %v", channels)
	if channels == nil {
		t.Fatalf("Unexpected empty Channels slice from GetSubscriptionsForChannelWithUid")
	}
	if len(channels) != 1 {
		t.Fatalf("Expected length of Channels slice 1, given %d", len(channels))	
	}
	ch := channels[0]
	if ch.Id != SUBSCRIBED_CHANNEL_ID {
		errMsgFormat := "Expected subscribed Channel with Id `%s`, given `%s`"
		t.Fatalf(errMsgFormat, SUBSCRIBED_CHANNEL_ID, ch.Id)	
	}
	if ch.Name != SUBSCRIBED_CHANNEL_NAME {
		errMsgFormat := "Expected subscribed Channel with name `%s`, given `%s`"
		t.Fatalf(errMsgFormat, SUBSCRIBED_CHANNEL_NAME, ch.Name)	
	}
	// Verify server was called as expected
	if !serverCalled {
		t.Fatalf("Expected to have posted Channel subscription data to graph database server")
	}
}