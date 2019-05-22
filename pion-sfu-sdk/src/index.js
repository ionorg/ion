/* global */
// eslint-disable-next-line
//
import { EventEmitter } from 'events';
import Room from './Room'
import RTC from './RTC'
import Player from './Player'

export default class SFU  extends EventEmitter {

    onRoomConnect = () => {
        console.log('onRoomConnect')
        this.join(true);

        this.players = new Map();
        this.rtc = new RTC();

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

        this.onCreateSender(this.room.peerId);
    }

    onRoomDisconnect = () => {
        console.log('onRoomDisconnect')
    }

    async onCreateSender(pubid) {
        try {
            let sender = await this.rtc.createSender();
            sender.pc.onicecandidate = async (e) => {
                if (!sender.senderOffer) {
                    var offer = sender.pc.localDescription;
                    console.log('Send offer sdp => ' + offer.sdp);
                    sender.senderOffer = true
    
                    let answer = await this.room.publish(offer,pubid);
                    console.log('Got answer(' + pubid + ') sdp => ' + answer.jsep.sdp);
                    sender.pc.setRemoteDescription(answer.jsep);
                }
            }
            let desc = await sender.pc.createOffer({ offerToReceiveVideo: false, offerToReceiveAudio: false })
            sender.pc.setLocalDescription(desc);
        }catch(error){
            console.log('onCreateSender error => ' + error);
        }
    }

    async onRtcCreateRecver(pubid) {
        try {
            let receiver = this.rtc.createRecver(pubid);
            receiver.pc.onicecandidate = async (e) => {
                if (!receiver.senderOffer) {
                    var offer = receiver.pc.localDescription;
                    console.log('Send offer sdp => ' + offer.sdp);
                    receiver.senderOffer = true
                    let answer = await this.room.subscribe(offer,pubid);
                    console.log('Got answer(' + pubid + ') sdp => ' + answer.sdp);
                    receiver.pc.setRemoteDescription(answer);
                }
            }
            let desc = await receiver.pc.createOffer();
            receiver.pc.setLocalDescription(desc);
        }catch(error){
            console.log('onRtcCreateRecver error => ' + error);
        }
    }

    onRtcLeaveRecver(pubid) {
        this.rtc.closeRecver(pubid)
    }

    constructor (roomId) {
        super()
        this.roomId = roomId;

        this.room = new Room(roomId)
        // bind event callbaks.
        this.room.on('onRoomConnect', this.onRoomConnect)
        this.room.on('onRoomDisconnect',this.onRoomDisconnect);
        this.room.on('onRtcCreateRecver', this.onRtcCreateRecver.bind(this))
        this.room.on('onRtcLeaveRecver', this.onRtcLeaveRecver.bind(this))
    }

    connect() {
        console.log('Connect to [' + this.roomId + ']')
    }

    close () {
        console.log('Force close.')
        this.room.close()
    }

    join (sender) {
        this.room.join(sender)
    }

    leave () {
        this.room.leave()
    }
}

window.SFU = SFU
