# protocol

the process is: 

```
client<----wss---->ion<---mq--->islb<---mq--->ion<----wss---->client
```



## client to ion
This protocol base on [protoo](https://protoojs.org/#protoo)

### 1) join room

#### request

```
method:join
data:{
    "rid": "$roomId",
    "info":{"name": "$name"}
}
```

#### response
```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1

```
Somebody will receive "stream-add"  if there are some publishers in this room

### 2) leave room
#### request

```
method:leave
data:{
    "rid":"$roomId"
}
```

#### response
```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```
Subscriber will receive "stream-remove" when  publisher leave this room

### 3) publish

#### request

```
method:publish
data:{
    "jsep": {"type": "offer","sdp": "..."},
    "options": {"codec": "H264", "video": true, "audio": true, "screen": false}
}
```
codec should be H264 or VP8

#### response
```
//success
ok:true
data:{
    "mid": "$mediaId"
    "jsep": {"type": "answer","sdp": "..."}
}

//failed
ok:false
errCode:-1
```

### 4) unpublish

####request

```
method:unpublish
data:{
    "rid": "$roomId",
    "mid": "$mediaId"
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

### 5) subscribe

#### request
```
method:subscribe
data:{
    "rid": "$roomId",
    "mid": "$mediaId",
    "jsep": {"type": "offer","sdp": "..."}
}
```

#### response
```
//success
ok:true
data:{
    "jsep": {"type": "answer","sdp": "..."}
}

//failed
ok:false
errCode:-1
```
### 6) unsubscribe

####request

```
method:unsubscribe
data:{
    "rid": "$roomId",
    "mid": "$mediaId"
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

## 7) broadcast

####request

```
method:broadcast
data:{
    "rid":"$roomId",
    "uid":"$uid",
    "info": {
        "name":"$name",
        "msg":"$message"
    }
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

## 

## Events from ion

### 1) peer-join

Ion will broadcast "peer-join" when someone joined this room

####request

```
method:peer-join
data:{
    "rid": "$rid"
    "id": "$id"
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

### 2) peer-leave

Ion will broadcast "peer-leave" when someone left from room

####request

```
method:peer-leave
data:{
    "rid": "$rid"
    "id": "$id"
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

### 3) stream-add

Ion will broadcast "stream-add" when someone published success

People can subscribe while receiving "stream-add" in this room

#### request

```
method:stream-add
data:{
    "rid": "$roomId",
    "mid": "$mediaId"
}
```

#### response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

### 4) stream-remove

Ion will broadcast "stream-remove" when publisher leave room

subscribers need to release resources(pc,player,etc.) when they receive "stream-remove"

####request

```
method:stream-remove
data:{
    "rid": "$roomId",
    "mid": "$mediaId"
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

## 5) broadcast

####request

```
method:broadcast
data:{
    "rid":"$roomId",
    "uid":"$uid",
    "info": {
        "name":"$name",
        "msg":"$message"
    }
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

##

## ion to islb

 Ion exchange message from islb by rabbitmq

### 1) peer-join

Ion will send "peer-join" when someone joined this room

Islb will broadcast this msg to all ions

####request

```
method:peer-join
data:{
    "rid": "$rid",
    "id": "$id",
    "info":{"name": "$name"}
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

### 2) peer-leave

Ion will send "peer-leave" when someone leave from room

Islb will broadcast this msg to all ions

####request

```
method:peer-leave
data:{
    "rid": "$rid"
    "id": "$id"
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

### 3) getPubs

Ion will  get all publiser's information from islb when somebody join a room

Islb will response all publisher's information one by one

Then ion can send these infomation to this new joiner by signal

#### request

```
{
    "method": "getPubs",
    "rid": "$roomId",
    "skipId": "$skipId"
}
```

#### response1
```
{
    "mid": "$mediaId1",
    "info": "$info1"
}

```

#### response2

```
{
    "mid": "$mediaId2",
    "info": "$info2"
}
```

### ...

### 4) stream-add

ion will send "stream-add" to islb when client start publishing

#### request

```
{
    "method": "stream-add",
    "rid": "$roomId",
    "mid": "$meidaId",
    "mediaInfo": {"$SSRC": $payloadType}
}
```

### 5) keepAlive

ion send "keepAlive" to islb periodically when publish stream success

#### request

```
{
    "method": "keepAlive",
    "rid": "$roomId",
    "mid": "$meidaId",
    "info": {"$SSRC": $payloadType}
}
```

### 4) stream-remove

Ion will send "stream-remove" to islb when client unpublish stream
#### request
```
{
    "method": "stream-remove",
    "rid": "$roomId",
    "mid": "$meidaId"
}
```

### 5) relay

Ion will send "relay" to islb when publisher is on other ion

#### request

```
{
    "method": "relay",
    "rid": "$roomId",
    "mid": "$meidaId"
}
```

### 6) getMediaInfo

Ion will send "getMediaInfo" to islb when publishing is on other ion

####request

```
{
    "method": "getMediaInfo",
    "rid": "$roomId",
    "mid": "$meidaId"
}
```

### 7) unRelay

Ion will send "unRelay" to islb when this rtp-pub has no sub, and islb will control origin ion to stop this relay

#### request

```
{
    "method": "unRelay",
    "rid": "$roomId",
    "mid": "$meidaId"
}
```

##8) broadcast

####request

```
method:broadcast
data:{
    "rid":"$roomId",
    "uid":"$uid",
    "info": {
        "name":"$name",
        "msg":"$message"
    }
}
```

####response

```
//success
ok:true
data:{}

//failed
ok:false
errCode:-1
```

##