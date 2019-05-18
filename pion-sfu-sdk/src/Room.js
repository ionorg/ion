/* global */
// eslint-disable-next-line
//
import { EventEmitter } from 'events';
import Signal from './Signal'

var url = 'wss://' + window.location.host + ':8000/connection/websocket'
if (window.location.host.search(':') !== -1) {
    url = 'wss://' + window.location.host.split(':')[0] + ':8000/connection/websocket'
}
const token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJyb29tMSIsImV4cCI6MTU4NjM0NTczNn0.p0NtUrooCBPZqpszXczoc7G8EQpCCSWmz-QstbxWeug'

class Room extends EventEmitter {
    constructor(roomid) {
        super()
        this.roomID = roomid
        this.signal = new Signal({
            url: url,
            token: token,
            roomid: roomid
        })
        this.signal.on('connect', this._onconnect)
        this.signal.on('disconnect', () => this.emit('onRoomDisconnect'))
        this.reqID = 1
    }

    /*
        {
            "result": {
                "channel": "signal:room1",
                "data": {
                    "data": {
                        "req": "leave",
                        "id": 4,
                        "msg": {
                            "client": "1b5731a3-2c3c-4726-9b7b-11ea94d63508"
                        }
                    },
                    "info": {
                        "user": "room1",
                        "client": "1b5731a3-2c3c-4726-9b7b-11ea94d63508"
                    }
                }
            }
        }
    */

    onClientSubscribe = (msg) => {

        console.log('Got message => ' + JSON.stringify(msg))

        // drop self message
        if (this.clientID === msg.info.client) {
            // console.log('skip self message')
            return
        }

        if (msg.data.req !== undefined) { //handle request
            const request = msg.data.req

            switch (request) {
                case 'onPublish': {
                    var pubid = msg.data.msg.pubid;
                    if (pubid === this.clientID)
                        return;
                    console.log('Got publish from => ' + pubid)
                    this.emit('onRtcCreateRecver', pubid)
                }
                break;
            case 'leave': {
                var pubid = msg.data.msg.client
                console.log('[' + pubid + ']  => leave !!!!')
                this.emit('onRtcLeaveRecver', pubid)
            }
            break;
            }

        } else if (msg.data.resp != undefined) { //handle response
            const response = msg.data.resp;
            if (response === 'success') {
                if (msg.data.msg.type !== undefined &&
                    msg.data.msg.type === 'sender' &&
                    this.senderOfferReqID === msg.data.id
                ) {
                    console.log('Room.onRtcSetSenderRemoteSDP ' + this.senderOfferReqID.toString())
                    this.emit('onRtcSetSenderRemoteSDP', msg.data.msg.jsep)
                } else if (msg.data.msg.pubid !== undefined && msg.data.msg.pubid !== this.clientID) {
                    console.log('Room.onRtcSetRecverRemoteSDP(' + msg.data.msg.pubid + ')')
                    this.emit('onRtcSetRecverRemoteSDP', msg.data.msg.pubid, msg.data.msg.jsep)
                }
            }
        }
    }

    _onconnect = (ctx) => {
        this.clientID = ctx.client
        console.log('Room.onconnect ', ctx)
        this.signal.subscribe(this.roomID, this.onClientSubscribe)
        this.signal.subscribe(this.clientID, this.onClientSubscribe)
        this.emit('onRoomConnect')
    }

    connect() {
        console.log('Room.connect')
        try {
            this.signal.connect()
        } catch (e) {
            console.log('Room.connect fail')
        }
    }

    disconnect() {
        this.signal.disconnect()
    }

    join(sender) {
        let joinMsg
        if (sender) {
            joinMsg = {
                'req': 'join',
                'id': this.reqID++,
                'msg': {
                    'client': this.clientID,
                    'type': 'sender'
                }
            }
        } else {
            joinMsg = {
                'req': 'join',
                'id': this.reqID++,
                'msg': {
                    'client': this.clientID,
                    'type': 'recver'
                }
            }
        }
        this.signal.broadcast(joinMsg)
    }

    leave() {
        var leaveMsg = {
            'req': 'leave',
            'id': this.reqID++,
            'msg': {
                'client': this.clientID
            }
        }
        this.signal.broadcast(leaveMsg)
    }

    offer(sdp, offertype, pubid) {
        if (offertype === 'sender') {
            this.senderOfferReqID = this.reqID
            console.log('Room.offer this.senderOfferReqID=', this.senderOfferReqID)
            var pubMsg = {
                'req': 'publish',
                'id': this.reqID,
                'msg': {
                    'type': offertype,
                    'jsep': sdp
                }
            }
            this.signal.publish(pubMsg)
        } else {
            //this.recverOfferReqID = this.reqID
            //console.log('Room.offer this.recverOfferReqID=', this.recverOfferReqID)
            console.log('Room.offer pubid=', pubid)
            var subMsg = {
                'req': 'subscribe',
                'id': this.reqID,
                'msg': {
                    'type': 'recver',
                    'pubid': pubid,
                    'jsep': sdp
                }
            }
            this.signal.publish(subMsg)
        }
        this.reqID++
    }
}

export default Room