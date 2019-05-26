import { EventEmitter } from 'events';

const ices = 'stun:stun.stunprotocol.org:3478'

export default class RTC extends EventEmitter {
    constructor() {
        super();
        this._sender = {};
        this._receivers = new Map();
    }

    get sender () {
        return this._sender;
    }

    getReceivers (pubid) {
        return this._receivers.get(pubid);
    }

    async createSender(pubid) {
        let sender = {
            offerSent: false,
            pc: null,
        };
        sender.pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] });
        let stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
        sender.pc.addStream(stream);
        this.emit('localstream', pubid, stream);
        this._sender = sender;
        return sender;
    }

    createRecver (pubid) {
        try {
            let receiver = {
                offerSent: false,
                pc: null,
                id: pubid,
                streams: []
            };

            var pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] });
            pc.onicecandidate = e => {
                console.log('receiver.pc.onicecandidate => ' + e.candidate)
            }

            pc.addTransceiver('audio', { 'direction': 'recvonly' });
            pc.addTransceiver('video', { 'direction': 'recvonly' });

            pc.onaddstream = (e) => {
                var stream = e.stream;
                console.log('receiver.pc.onaddstream', stream.id);
                var receiver = this._receivers.get(pubid);
                receiver.streams.push(stream);
                this.emit('addstream', pubid, stream);
            }

            pc.onremovestream = (e) => {
                var stream = e.stream;
                console.log('receiver.pc.onremovestream', stream.id);
                this.emit('removestream', pubid, stream);
            }
            receiver.pc = pc;
            this._receivers.set(pubid,receiver);
            return receiver;
        } catch (e) {
            console.log(e);
            throw e;
        }
    }

    closeRecver(pubid) {
        var receiver = this._receivers.get(pubid)
        if(receiver) {
            receiver.streams.forEach(stream => {
                this.emit('removestream', pubid, stream)
            })
            receiver.pc.close();
            this._receivers.delete(pubid);
        }
    }
}
