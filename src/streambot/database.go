package streambot

import(
	"errors"
	"fmt"
)
import rexster "github.com/sqs/go-rexster-client"

type Database interface {
	SaveChannel(ch *Channel) (err error)
	GetChannelWithUid(uid string) (err error, ch *Channel)
	SaveChannelSubscription(fromChannelId string, toChannelId string, creationTime int64) (err error)
	GetSubscriptionsForChannelWithUid(uid string) (err error, chs []Channel)
}

/* At time of development there is a specialy to consider about the rexster backend server. As 
 * rexster runs within the Titan+Cassandra server distribution there is limitation of it using 
 * TitanGraphConfiguration that doesn't support manual indices and setting of vertex or edge IDs.
 * Titan creates those IDs and any delivered with creation request is ignored.
 * To keep a unique identifier index on vertices another property named `uid` is supposed to 
 * capture that ID and persist in Titan-Cassandra. */

type GraphDatabase struct {
	Graph rexster.Graph
}

func NewGraphDatabase(graph_name string, host string, port uint16) (db *GraphDatabase) {
	var r = rexster.Rexster{host, port, false}
	var g = rexster.Graph{graph_name, r}
	db = &GraphDatabase{g}
	return
}

func (db *GraphDatabase) SaveChannel(ch *Channel) (err error) {
	// Create a vertex in the graph database for the channel
	var properties = map[string]interface{}{"name": ch.Name, "uid": ch.Id}
	vertex := rexster.NewVertex("", properties)
	_, err = db.Graph.CreateOrUpdateVertex(vertex)
	return
}

func (db *GraphDatabase) GetChannelWithUid(uid string) (err error, ch *Channel) {
	res, err := db.Graph.QueryVertices("uid", uid)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to query vertices at Rexster:", err))
		return
	}
	if vs := res.Vertices(); vs != nil {
		numVertices := len(vs)
		if numVertices > 1 {
			errMsgFormat := "Unexpectedly Rexster backend returned more than one vertex, given `%v`"
			errMsg := fmt.Sprintf(errMsgFormat, vs)
			err = errors.New(errMsg)
		} else if numVertices == 1 {
			vertex := vs[0]
			ch = &Channel{vertex.Map["uid"].(string), vertex.Map["name"].(string)}
		}
	} else {
		errMsgFormat := "Unexpectedly Rexster backend returned no vertex, given `%v`"
		errMsg := fmt.Sprintf(errMsgFormat, res)
		err = errors.New(errMsg)
	}
	return
}

func (db *GraphDatabase) SaveChannelSubscription(
	fromChannelId string, 
	toChannelId string, 
	creationTime int64,
) (err error) {
	script := fmt.Sprintf(
		"subs=g.V('uid', '%s')" +
        	".out('subscribe').has('id', g.V('uid', '%s').next().id);" +
        	"if(!subs.hasNext()){" +
        	"e=g.addEdge(g.V('uid','%s').next(),g.V('uid','%s').next()," +
        	"'subscribe',[time:%d]);g.commit();e" +
			"}else{g.V('uid', '%s').outE('subscribe')}",
		fromChannelId, 
		toChannelId, 
		fromChannelId, 
		toChannelId, 
		creationTime,
		fromChannelId,
	)
	_, err = db.Graph.Eval(script)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to query vertices at Rexster:", err))
		return
	}
	return
}

func (db *GraphDatabase) GetSubscriptionsForChannelWithUid(uid string) (err error, chs []Channel) {
	script := fmt.Sprintf("g.V(\"uid\",\"%s\").out.loop(1){it.loops < 100}{true}.dedup", uid)
	res, err := db.Graph.Eval(script)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to query subscribed channels at Rexster:", err))
	}
	if vs := res.Vertices(); vs != nil {
		numVertices := len(vs)
		if numVertices > 0 {
			chs = make([]Channel, numVertices)
			for idx, vertex := range vs {
				chs[i] = Channel{vertex.Map["uid"].(string), vertex.Map["name"].(string)}	
			}
		}
	} else {
		errMsgFormat := "Unexpectedly Rexster backend returned no vertex for channel " +
		"subscription, given `%v`"
		errMsg := fmt.Sprintf(errMsgFormat, res)
		err = errors.New(errMsg)
	}
	return
}