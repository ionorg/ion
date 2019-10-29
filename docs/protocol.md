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
    "rid":"$roomid"
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
when somebody join success, ion broadcast "onPublish" to him if there are some publishers

### 2) leave room
#### request

```
method:leave
data:{
    "rid":"$roomid"
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
### 3) publish
#### request

```
method:publish
data:{
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

### 4) onPublish

when publisher published success, ion broadcast "onPublish" to others
#### request

```
method:onPublish
data:{
    "pubid": "$pubid"
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

### 5) subscribe

client could subscribe $pubid when it got "onPublish"
#### request
```
method:subscribe
data:{
    "pubid:"$pubid",
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
### 6) onUnpublish

when publisher leave room, ion broadcast "onUnpublish"

subscribers need to delete this pc and player when they receive "onUnpublish"
####request
```
method:onUnpublish
data:{
    "pubid": "$pubid"
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

### 7) unpublish

when publisher unpublish, SFU broadcast "onUnpublish"

subscribers need to delete this pc and player when they receive "onUnpublish"
####request
```
method:unpublish
data:{

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

### 7) unsubscribe

####request
```
method:unsubscribe
data:{
    "pubid": "$pubid"
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



## ion to islb

 ion send message to islb by rabbitmq

### 1) getPubs

when somebody join a room, ion will send "getPubs" to islb
#### request

```
{
    "method": "getPubs",
    "rid": "$roomid",
    "pid": "$pubid"
}
```

#### response
```
{
    "pid": "$pubid1",
    "info": "$info1"
}

{
    "pid": "$pubid2",
    "info": "$info2"
}
...
```

### 2) unpublish

ion will send "unpublish" to islb when client end publishing
#### request
```
{
    "method": "unpublish",
    "rid": "$roomid",
    "pid": "$pid"
}
```

#### 
ion will send "unRelay" to islb when this rtp-pub has no sub, and islb will control other ion to stop this relay

#### request

```
{
    "method": "unRelay",
    "rid": "$roomid",
    "pid": "$pid"
}
```

### 3) publish

ion will send "publish" to islb when client start publishing

#### request

```
{
    "method": "publish",
    "rid": "$roomid",
    "pid": "$pid",
    "info": {"$SSRC": $payloadType}
}
```

### 4) keepAlive

ion send "keepAlive" to islb, islb will keep this status alive in redis

#### request

```
{
    "method": "keepAlive",
    "rid": "$roomid",
    "pid": "$pid",
    "info": {"$SSRC": $payloadType}
}
```

### 5) relay

ion send "relay" to islb when publishing is on other ion

#### request

```
{
    "method": "relay",
    "rid": "$roomid",
    "pid": "$pid"
}
```

ion send "getMediaInfo" to islb when publishing is on other ion

####request

```
{
    "method": "getMediaInfo",
    "rid": "$roomid",
    "pid": "$pid"
}
```

