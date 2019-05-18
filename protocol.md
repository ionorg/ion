
## 1. public channel


There are one public channel `signal:$roomid`.

All clients should subscribe it, SFU will know which one join/leave this room

## 2. private channel

SFU get a `$clientid` when client join a room.

Client and SFU must subscribe to the channel(`$clientid`).

Now they can publish offer and answer to this channel.

`$reqid` is a request id, so client and SFU can match each request and response.

recently, `$reqid` = `$i++`(i=1)
## 3. protocol
## 1) join room
### publish to channel: `signal:$roomid`
$type: sender/recver

```
{ "req":"join", "id":$reqid, "msg":{"client":"$clientid", "type":"$type"}}

//success
{ "resp":"success", "id":$reqid, "msg":{}}

//failed
{ "resp":"failed", "id":$reqid, "msg":{}}
```

## 2) leave room
###publish to channel: `signal:$roomid`

```
{ "req":"leave", "id":$reqid, "msg":{"client":"$clientid"}}

//success
{ "resp":"success", "id":$reqid, "msg":{}}

//failed
{ "resp":"failed", "id":$reqid, "msg":{}}
```


## 3) publish
publish to channel: **$clientid**

```
{
    "req": "publish",
    "id": $reqid,
    "msg": {
        "type": "sender",
        "jsep": {"type": "offer","sdp": "..."}
    }
}

//success
{
    "resp": "success",
    "id": $reqid,
    "msg": {
        "type": "sender",
        "jsep": {"type": "answer","sdp": "..."}
    }
}

//failed
{
    "resp": "failed",
    "id": $reqid,
    "msg": {

    }
}

```

## 4) onPublish

when publisher published success, SFU broadcast "onPublish"


```
{
    "req": "onPublish",
    "id": 0,
    "msg": {
        "type": "sender",
        "pubid": "$pubid"
    }
}

//success
{
    "resp": "success",
    "id": 0,
    "msg": {
    }
}

//failed
{
    "resp": "failed",
    "id": 0,
    "msg": {

    }
}

```


## 5) subscribe

client can subscribe $pubid when it get "onPublish"

```
{
    "req": "subscribe",
    "id": "$reqid",
    "msg": {
        "type": "recver",
        "pubid": "$pubid",
        "jsep": {"type": "offer","sdp": "..."}
    }
}

//success
{
    "resp": "success",
    "id": "$reqid",
    "msg": {
        "jsep": {"type": "answer","sdp": "..."}
    }
}

//failed
{
    "resp": "failed",
    "id": "$reqid",
    "msg": {

    }
}
```

## 6) onUnpublish

when publisher leave room, SFU broadcast "onUnpublish"

subscribers need to delete this pc and player when they receive "onUnpublish"


```
{
    "req": "onUnpublish",
    "id": 0,
    "msg": {
        "pubid": "$pubid"
    }
}

//success
{
    "resp": "success",
    "id": 0,
    "msg": {
    }
}

//failed
{
    "resp": "failed",
    "id": 0,
    "msg": {

    }
}

```
## 7) control [WIP]
publishers must publish their devices information, like "muted" "close camera"..