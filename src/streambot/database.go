package streambot

import(
	"errors"
	"fmt"
)
import rexster "github.com/mbiermann/go-rexster-client"

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

func NewGraphDatabase(graph_name string, hosts []string) (db *GraphDatabase, err error) {
	r, err := rexster.NewRexster(&rexster.RexsterOptions{
		Hosts: hosts,
		Debug: true,
		NodeReanimationAfterSeconds: 300,
	})
	if err != nil {
		errMsgFormat := "Unexpected error when intializing Rexster cluster client: %v"
		err = errors.New(fmt.Sprintf(errMsgFormat, err))
		return
	}
	var g = rexster.Graph{graph_name, *r}
	db = &GraphDatabase{g}
	return
}

func (db *GraphDatabase) SaveChannel(ch *Channel) (err error) {
	// Create a vertex in the graph database for the channel
	var properties = map[string]interface{}{"name": ch.Name, "uid": ch.Id}
	vertex := rexster.NewVertex("", properties)
	_, err = db.Graph.CreateOrUpdateVertex(vertex)
	fmt.Println(fmt.Sprintf("Vertex is %v", vertex))
	if err != nil {
		errMsgFormat := "Unexpected error when saving Channel vertex `%v`: %v"
		err = errors.New(fmt.Sprintf(errMsgFormat, vertex, err))
	}
	return
}

func GetVertexWithUid(db *GraphDatabase, uid string) (v *rexster.Vertex, err error) {
	res, err := db.Graph.QueryVertices("uid", uid)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to query vertices at Rexster with error: %v", err))
		return
	}
	if res == nil {
		err = errors.New(fmt.Sprintf("Rexster backend did not respond"))
		return
	}
	if vs := res.Vertices(); vs != nil {
		numVertices := len(vs)
		if numVertices > 1 {
			errMsgFormat := "Unexpectedly Rexster backend returned more than one vertex, given `%v`"
			errMsg := fmt.Sprintf(errMsgFormat, vs)
			err = errors.New(errMsg)
		} else if numVertices == 1 {
			v = vs[0]
		} else {
			errMsgFormat := "Unexpectedly Rexster backend returned no vertex, given `%v`"
			errMsg := fmt.Sprintf(errMsgFormat, res)
			err = errors.New(errMsg)
		}
	}
	return
}

func (db *GraphDatabase) GetChannelWithUid(uid string) (err error, ch *Channel) {
	vertex, err := GetVertexWithUid(db, uid)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to query vertices at Rexster with error: %v", err))
		return
	}
	ch = &Channel{vertex.Map["uid"].(string), vertex.Map["name"].(string)}
	return
}

func (db *GraphDatabase) SaveChannelSubscription(
	fromChannelId string, 
	toChannelId string, 
	creationTime int64,
) (err error) {
	var outV *rexster.Vertex
	outV, err = GetVertexWithUid(db, fromChannelId)
	fmt.Printf("OutV %v", outV)
	if err != nil {
		errMsgFormat := "Unexpected error when querying Channel vertex with ID `%s`: %v"
		err = errors.New(fmt.Sprintf(errMsgFormat, fromChannelId, err))
		return
	}
	if outV == nil {
		errMsgFormat := "Unexpected missing Channel vertex with ID `%s`"
		err = errors.New(fmt.Sprintf(errMsgFormat, fromChannelId))
		return	
	}
	inV, err := GetVertexWithUid(db, toChannelId)
	if err != nil {
		errMsgFormat := "Unexpected error when querying Channel vertex with ID `%s`: %v"
		err = errors.New(fmt.Sprintf(errMsgFormat, toChannelId, err))
		return
	}
	if inV == nil {
		errMsgFormat := "Unexpected missing Channel vertex with ID `%s`"
		err = errors.New(fmt.Sprintf(errMsgFormat, toChannelId))
		return	
	}
	fmt.Printf("InV %v", inV)
	edge := rexster.NewEdge("", outV.Id(), "subscribe", inV.Id(), map[string]interface{}{"time": creationTime})
	_, err = db.Graph.CreateOrUpdateEdge(edge)
	if err != nil {
		errMsgFormat := "Unexpected error when saving Channel Subscription edge `%v`: %v"
		err = errors.New(fmt.Sprintf(errMsgFormat, edge, err))
	}
	return
}

func (db *GraphDatabase) GetSubscriptionsForChannelWithUid(uid string) (err error, chs []Channel) {
	script := fmt.Sprintf("g.V(\"uid\",\"%s\").out.loop(1){it.loops < 50}{true}.dedup", uid)
	res, err := db.Graph.Eval(script)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to query subscribed channels at Rexster:", err))
		return
	}
	if res == nil {
		err = errors.New(fmt.Sprintf("Rexster backend did not respond"))
		return
	}
	if vs := res.Vertices(); vs != nil {
		numVertices := len(vs)
		if numVertices > 0 {
			chs = make([]Channel, numVertices)
			for idx, vertex := range vs {
				chs[idx] = Channel{vertex.Map["uid"].(string), vertex.Map["name"].(string)}	
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