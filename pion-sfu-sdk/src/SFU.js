/* global */
// eslint-disable-next-line
//
import { EventEmitter } from 'events';
import Room from './Room'
import RTC from './RTC'
import Player from './Player'

export default class SFU  extends EventEmitter {

    constructor () {
        super()
        this.room = new Room()
        // bind event callbaks.
        this.room.on('onRoomConnect', this.onRoomConnect)
        this.room.on('onRoomDisconnect',this.onRoomDisconnect);
        this.room.on('onRtcCreateRecver', this.onRtcCreateRecver.bind(this))
        this.room.on('onRtcLeaveRecver', this.onRtcLeaveRecver.bind(this))
        this.room.on('onCreateSender',this.onCreateSender.bind(this))
    }

    close () {
        console.log('Force close.')
        this.room.close()
    }

    join (roomId) {
        console.log('Join to [' + roomId + ']')
        this.room.join(roomId)
    }

    publish () {
        this.onCreateSender(this.room.peerId)
    }

    leave () {
        this.room.leave()
    }

    onRoomConnect = () => {
        console.log('onRoomConnect')
        this.rtc = new RTC();

        this.rtc.on('localstream',(id, stream) => {
            this.emit('addLocalStream', id, stream);
        })

        this.rtc.on('addstream',(id, stream) => {
            this.emit('addRemoteStream', id, stream);
        })

        this.rtc.on('removestream',(id, stream) => {
            this.emit('removeRemoteStream', id, stream);
        })

        this.emit('connect');
    }

    onRoomDisconnect = () => {
        console.log('onRoomDisconnect')
        this.emit('disconnect');
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
}