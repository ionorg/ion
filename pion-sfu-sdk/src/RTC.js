/* global  RTCPeerConnection,  */
// eslint-disable-next-line
//
import { EventEmitter } from 'events';

const ices = 'stun:stun.stunprotocol.org:3478'

export default class RTC extends EventEmitter {
    constructor() {
        super()
        this.sendPC = null;
        this.receivers = new Map();
    }

    createSender(pubid) {
        this.sendPC = new RTCPeerConnection({ iceServers: [{ urls: ices }] })
        this.sendPC.oniceconnectionstatechange = e => {
            // console.log('sendPC.oniceconnectionstatechange' + e)
        }
        this.sendPC.onicecandidate = e => {
            if (!this.senderOffer) {
                var offer = this.sendPC.localDescription;
                console.log('Send offer => ' + offer.sdp)
                this.emit('offer', offer, 'sender', pubid)
                this.senderOffer = true
            }
        }
        this.sendPC.onnegotiationneeded = e => {
        }
        navigator.mediaDevices.getUserMedia({ video: true, audio: true }).then((stream) => {
            this.sendPC.addStream(stream)
            this.sendPC.createOffer({ offerToReceiveVideo: false, offerToReceiveAudio: false })
                .then(desc => {
                    this.sendPC.setLocalDescription(desc)
                })
            this.emit('localstream', pubid, stream)
        })
    }

    createRecver (pubid) {
        try {
            var recvPC = new RTCPeerConnection({ iceServers: [{ urls: ices }] })
            recvPC.onicecandidate = e => {
                console.log('recvPC.onicecandidate => ' + e.candidate)
            }

            recvPC.addTransceiver('audio', { 'direction': 'recvonly' })
            recvPC.addTransceiver('video', { 'direction': 'recvonly' })

            recvPC.createOffer()
                .then(desc => {
                    recvPC.setLocalDescription(desc)
                    console.log('createOffer(' + pubid + ') sdp => ' + desc.sdp)
                    this.emit('offer', recvPC.localDescription, 'recver', pubid);
                })

            recvPC.onaddstream = (e) => {
                var stream = e.stream;
                console.log('recvPC.onaddstream', stream.id)
                var receiver = this.receivers.get(pubid)
                receiver.streams.push(stream)
                this.emit('addstream', pubid, stream)
            }

            recvPC.onremovestream = (e) => {
                var stream = e.stream
                console.log('recvPC.onremovestream', stream.id)
                this.emit('removestream', pubid, stream)
            }

            var receiver = {
                pc: recvPC,
                id: pubid,
                streams: []
            }

            this.receivers.set(pubid,receiver);
        } catch (e) {
            console.log(e)
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
        return this.sendPC
    }

    getRecver (pubid) {
        return this.receivers.get(pubid);
    }
}
