import { EventEmitter } from 'events';
import Room from './Room'
import RTC from './RTC'
import Player from './Player'

export default class SFU  extends EventEmitter {

    onRoomConnect = () => {
        console.log('SFU.onRoomConnect')
        this.players = new Map();
        this.join(true)
        this.rtc = new RTC()


        this.rtc.on('localstream',(id, stream) => {
            var player = new Player({ id, stream });
            this.players.set(id,player)
            console.log('Create Player(' + stream.id + ') for Local stream!')
        })

        this.rtc.on('addstream',(id, stream) => {
            var player = new Player({ id, stream });
            this.players.set(id,player)
            console.log('Create Player(' + stream.id + ') for Remote stream!')
        })

        this.rtc.on('removestream',(id, stream) => {
            var player = this.players.get(id);
            if(player){
                player.destroy();
                this.players.delete(id);
                console.log('Close Player(' + stream.id + ') for Remote stream!')
            }
        })

        this.rtc.on('offer',(sdp, type , pubid) => {
            this.room.offer(sdp, type, pubid);
        });

        this.rtc.createSender(1)
    }

    onRoomDisconnect = () => {
        console.log('onRoomDisconnect')
    }

    onRtcCreateRecver = (pubid) => {
        this.rtc.createRecver(pubid)
    }

    onRtcLeaveRecver = (pubid) => {
        this.rtc.closeRecver(pubid)
    }

    onRtcSetSenderRemoteSDP = (sdp) => {
        console.log('onRtcSetSenderRemoteSDP====', sdp)
        this.rtc.getSender().setRemoteDescription(sdp)
    }

    onRtcSetRecverRemoteSDP = (pubid, sdp) => {
        console.log('onRtcSetRecverRemoteSDP(' + pubid + ')==== \n ' + sdp )
        var receiver = this.rtc.getRecver(pubid);
        if(receiver)
            receiver.pc.setRemoteDescription(sdp)
    }

    constructor (roomId) {
        super()

        this.room = new Room(roomId)

        // bind event callbaks.
        this.room.on('onRoomConnect', this.onRoomConnect)
        this.room.on('onRoomDisconnect',this.onRoomDisconnect);
        this.room.on('onRtcCreateRecver', this.onRtcCreateRecver)
        this.room.on('onRtcLeaveRecver', this.onRtcLeaveRecver)
        this.room.on('onRtcSetSenderRemoteSDP',this.onRtcSetSenderRemoteSDP)
        this.room.on('onRtcSetRecverRemoteSDP', this.onRtcSetRecverRemoteSDP)
    }

    connect () {
        console.log('SFU.connect')
        this.room.connect()
    }

    join (sender) {
        this.room.join(sender)
    }

    leave () {
        this.room.leave()
    }
}

window.SFU = SFU
