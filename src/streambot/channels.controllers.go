package streambot

import(  
  "io/ioutil"
  "encoding/json"
  "github.com/laurent22/ripple"
  "github.com/op/go-logging"
  "time"
)

var log = logging.MustGetLogger("streambot-api")

type ChannelController struct {
  Database  Database
  Stats     *Statter
}

func NewChannelController(db Database, stats *Statter) *ChannelController {
  return &ChannelController{db, stats}
}

type PutChannelOutData struct {
  Id string `json:"id"`
}

type PutChannelInData struct {
  Name string `json:"name"`
}

func(ctrl *ChannelController) Put(ctx *ripple.Context) {
  ctrl.Stats.Count("channels.put")
  // Read the request into a raw buffer and unmarshal buffer to further handle request
  body, err := ioutil.ReadAll(ctx.Request.Body)
  if err != nil {
    ctx.Response.Status = 501
    errMsgFormat := "Unexpected error when read request body of Channel PUT: %v"
    log.Error(errMsgFormat, err)
    return
  }
  var req PutChannelInData
  err = json.Unmarshal(body, &req)
  if err != nil {
    ctx.Response.Status = 400
    errMsgFormat := "Unexpected error when parse Channel PUT request body `%v` at Rexster " +
    "backend: %v"
    log.Error(errMsgFormat, string(body), err)
    return
  }
  ch := NewChannel(req.Name)
  // Track timestamps in nanosecond precision before and after the database call
  beforeDB := time.Now()
  err = ctrl.Database.SaveChannel(ch)
  afterDB := time.Now()
  // Calculate database call duration and track in statter
  duration := afterDB.Sub(beforeDB)/time.Millisecond
  log.Debug("Database call SaveChannel in channels.Put took %d", duration)
  if err == nil {
    ctrl.Stats.Time("db.SaveChannel", int(duration))
  } else {
    // Evaluate errors
    ctx.Response.Status = 501
    log.Error("Database controller returned unexpected error on save Channel `%v`: %v", ch, err)
    return
  }
  ctx.Response.Body = PutChannelOutData{ch.Id}
}

type GetChannelOutData struct {
  Id    string `json:"id"`
  Name  string `json:"name"`
}

func(ctrl *ChannelController) Get(ctx *ripple.Context) {
  ctrl.Stats.Count("channels.get")
  id := ctx.Params["id"]
  if id == "" {
    ctx.Response.Status = 501
    log.Error("Missing Id on Channel GET")
    return
  }
  // Track timestamps in nanosecond precision before and after the database call
  beforeDB := time.Now()
  err, ch := ctrl.Database.GetChannelWithUid(id)
  afterDB := time.Now()
  // Calculate database call duration and track in statter
  duration := afterDB.Sub(beforeDB)/time.Millisecond
  log.Debug("Database call GetChannelWithUid in channels.Get took %d", duration)
  ctrl.Stats.Time("db.GetChannelWithUid", int(duration))
  if err != nil {
    ctx.Response.Status = 501
    log.Error("Unexpected error when fetch Channel with Id `%s` at Rexster backend: %v", id, err)
    return
  }
  if ch == nil {
    ctx.Response.Status = 400
    errMsgFormat := "Unexpected empty Channel when fetch Channel with Id `%s` at Rexster backend"
    log.Error(errMsgFormat, id)
    return
  }
  ctx.Response.Body = GetChannelOutData{ch.Id, ch.Name}
}

type PostChannelSubscriptionsInData struct {
  ToChannelId   string  `json:"channel_id"`
  Time          int64   `json:"created_at"`
}

func(ctrl *ChannelController) PostSubscriptions(ctx *ripple.Context) {
  ctrl.Stats.Count("channels.subscriptions.post")
  fromChannelId := ctx.Params["id"]
  if fromChannelId == "" {
    ctx.Response.Status = 400
    log.Error("Missing Id on Channel POST")
    return
  }
  // Read the request into a raw buffer and unmarshal buffer to post handle request
  body, err := ioutil.ReadAll(ctx.Request.Body)
  if err != nil {
    ctx.Response.Status = 400
    errMsgFormat := "Unexpected error when read request body of Channel POST with Id `%s`: %v"
    log.Error(errMsgFormat, fromChannelId, err)
    return
  }
  var req PostChannelSubscriptionsInData
  err = json.Unmarshal(body, &req)
  if err != nil {
    ctx.Response.Status = 400
    errMsgFormat := "Unexpected error when parse request body `%s` of Channel POST with Id `%s`: %v"
    log.Error(errMsgFormat, string(body), fromChannelId, err)
    return
  }
  // Track timestamps in nanosecond precision before and after the database call
  beforeDB := time.Now()
  err = ctrl.Database.SaveChannelSubscription(fromChannelId, req.ToChannelId, req.Time)
  afterDB := time.Now()
  // Calculate database call duration and track in statter
  duration := afterDB.Sub(beforeDB)/time.Millisecond
  log.Debug("Database call SaveChannelSubscription in channels.PostSubscriptions took %d", duration)
  ctrl.Stats.Time("db.SaveChannelSubscription", int(duration))
  if err != nil {
    ctx.Response.Status = 501
    errMsgFormat := "Database controller returned unexpected error on save subscription from " +
    "Channel with Id `%s` to Channel with Id `%s`, happend on `%d`: %v"
    log.Error(errMsgFormat, fromChannelId, req.ToChannelId, req.Time, err)
    return
  }
  ctx.Response.Status = 200
}

func(ctrl *ChannelController) GetSubscriptions(ctx *ripple.Context) {
  ctrl.Stats.Count("channels.subscriptions.get")
  id := ctx.Params["id"]
  if id == "" {
    ctx.Response.Status = 501
    log.Error("Missing Channel Id when fetch subscriptions")
    return
  }
  // Track timestamps in nanosecond precision before and after the database call
  beforeDB := time.Now()
  err, chs := ctrl.Database.GetSubscriptionsForChannelWithUid(id)
  afterDB := time.Now()
  // Calculate database call duration and track in statter
  duration := afterDB.Sub(beforeDB)/time.Millisecond
  log.Debug("Database call GetSubscriptionsForChannelWithUid in channels.GetSubscriptions took %d", duration)
  ctrl.Stats.Time("db.GetSubscriptionsForChannelWithUid", int(duration))
  if err != nil {
    ctx.Response.Status = 501
    errMsgFormat := "Unexpected error when fetch Channel subscriptions for Channel with Id `%s` " +
    "at Rexster backend: %v"
    log.Error(errMsgFormat, id, err)
    return
  }
  if chs == nil {
    ctx.Response.Status = 400
    errMsgFormat := "Unexpected empty Channels list when fetch Channel subscriptions for Channel" +
    " with Id `%s` at Rexster backend"
    log.Error(errMsgFormat, id)
    return
  }
  outChs := make([]GetChannelOutData, len(chs))
  for i := range chs {
    outChs[i] = GetChannelOutData{chs[i].Id, chs[i].Name}
  }
  ctx.Response.Body = outChs
}