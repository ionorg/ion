/* global  RTCPeerConnection,  */
// eslint-disable-next-line
//
import { EventEmitter } from 'events';

const ices = 'stun:stun.stunprotocol.org:3478'

export default class RTC extends EventEmitter {
    constructor() {
        super()
        this.sender = {};
        this.receivers = new Map();
    }

    async createSender(pubid) {
        let sender = {
            offerSent: false,
            pc: null,
        }
        sender.pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] })
        let stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true })
        sender.pc.addStream(stream);
        this.emit('localstream', pubid, stream)
        this.sender = sender;
        return sender;
    }

    createRecver (pubid) {
        try {
            let receiver = {
                offerSent: false,
                pc: null,
                id: pubid,
                streams: []
            }

            var pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] })
            pc.onicecandidate = e => {
                console.log('receiver.pc.onicecandidate => ' + e.candidate)
            }

            pc.addTransceiver('audio', { 'direction': 'recvonly' })
            pc.addTransceiver('video', { 'direction': 'recvonly' })

            pc.onaddstream = (e) => {
                var stream = e.stream;
                console.log('receiver.pc.onaddstream', stream.id)
                var receiver = this.receivers.get(pubid)
                receiver.streams.push(stream)
                this.emit('addstream', pubid, stream)
            }

            pc.onremovestream = (e) => {
                var stream = e.stream
                console.log('receiver.pc.onremovestream', stream.id)
                this.emit('removestream', pubid, stream)
            }
            receiver.pc = pc;
            this.receivers.set(pubid,receiver);
            return receiver;
        } catch (e) {
            console.log(e);
            throw e;
        }
    }

    closeRecver(pubid) {
        var receiver = this.getRecver(pubid)
        if(receiver) {
            receiver.streams.forEach(stream => {
                this.emit('removestream', pubid, stream)
            })
            receiver.pc.close();
            this.receivers.delete(pubid);
        }
    }

    getSender () {
        return this.sender
    }

    getRecver (pubid) {
        return this.receivers.get(pubid);
    }
}
