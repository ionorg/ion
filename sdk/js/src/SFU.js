import { EventEmitter } from 'events';
import Room from './Room'
import RTC from './RTC'

export default class SFU  extends EventEmitter {

    constructor () {
        super();
        this._room = new Room();
        this._rtc = new RTC();
        // bind event callbaks.
        this._room.on('onRoomConnect', this._onRoomConnect);
        this._room.on('onRoomDisconnect',this._onRoomDisconnect);
        this._room.on('onRtcCreateRecver', this._onRtcCreateRecver.bind(this));
        this._room.on('onRtcLeaveRecver', this._onRtcLeaveRecver.bind(this));
    }

    close () {
        console.log('Force close.');
        this._room.close();
    }

    join (roomId) {
        console.log('Join to [' + roomId + ']');
        this._room.join(roomId);
    }

    publish () {
        this._createSender(this._room.uid);
    }

    leave () {
        this._room.leave();
    }

    _onRoomConnect = () => {
        console.log('onRoomConnect');

        this._rtc.on('localstream',(id, stream) => {
            this.emit('addLocalStream', id, stream);
        })

        this._rtc.on('addstream',(id, stream) => {
            this.emit('addRemoteStream', id, stream);
        })

        this._rtc.on('removestream',(id, stream) => {
            this.emit('removeRemoteStream', id, stream);
        })

        this.emit('connect');
    }

    _onRoomDisconnect = () => {
        console.log('onRoomDisconnect');
        this.emit('disconnect');
    }

    async _createSender(pubid) {
        try {
            let sender = await this._rtc.createSender(pubid);
            sender.pc.onicecandidate = async (e) => {
                if (!sender.senderOffer) {
                    var offer = sender.pc.localDescription;
                    console.log('Send offer sdp => ' + offer.sdp);
                    sender.senderOffer = true
    
                    let answer = await this._room.publish(offer,pubid);
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

    async _onRtcCreateRecver(pubid) {
        try {
            let receiver = this._rtc.createRecver(pubid);
            receiver.pc.onicecandidate = async (e) => {
                if (!receiver.senderOffer) {
                    var offer = receiver.pc.localDescription;
                    console.log('Send offer sdp => ' + offer.sdp);
                    receiver.senderOffer = true
                    let answer = await this._room.subscribe(offer,pubid);
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

    _onRtcLeaveRecver(pubid) {
        this._rtc.closeRecver(pubid);
    }
}