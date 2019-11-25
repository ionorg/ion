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
Somebody will receive "stream-add"  if there are some publishers in this room

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
Subscriber will receive "stream-remove" when  publisher leave this room

### 3) publish

#### request

```
method:publish
data:{
    "jsep": {"type": "offer","sdp": "..."}
    "options": {"codec": "h264"}
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

### 4) unpublish

####request

```
method:unpublish
data:{
    "rid":"$roomid"
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
    "pid:"$pid",
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
    "pid": "$pid"
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
    "pid": "$pid"
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
    "pid": "$pid"
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
    "rid": "$roomid",
    "pid": "$pid"
}
```

#### response1
```
{
    "pid": "$pid1",
    "info": "$info1"
}

```

#### response2

```
{
    "pid": "$pid2",
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
    "rid": "$roomid",
    "pid": "$pid",
    "info": {"$SSRC": $payloadType}
}
```

### 5) keepAlive

ion send "keepAlive" to islb periodically when publish stream success

#### request

```
{
    "method": "keepAlive",
    "rid": "$roomid",
    "pid": "$pid",
    "info": {"$SSRC": $payloadType}
}
```

### 4) stream-remove

Ion will send "stream-remove" to islb when client unpublish stream
#### request
```
{
    "method": "stream-remove",
    "rid": "$roomid",
    "pid": "$pid"
}
```

### 5) relay

Ion will send "relay" to islb when publisher is on other ion

#### request

```
{
    "method": "relay",
    "rid": "$roomid",
    "pid": "$pid"
}
```

### 6) getMediaInfo

Ion will send "getMediaInfo" to islb when publishing is on other ion

####request

```
{
    "method": "getMediaInfo",
    "rid": "$roomid",
    "pid": "$pid"
}
```

### 7) unRelay

Ion will send "unRelay" to islb when this rtp-pub has no sub, and islb will control origin ion to stop this relay

#### request

```
{
    "method": "unRelay",
    "rid": "$roomid",
    "pid": "$pid"
}
```


