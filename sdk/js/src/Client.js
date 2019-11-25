import { EventEmitter } from 'events';
import protooClient from 'protoo-client';
import uuidv4 from 'uuid/v4';
import Stream from './Stream';
import * as sdpTransform from 'sdp-transform';

const ices = 'stun:stun.stunprotocol.org:3478';

const DefaultPayloadTypePCMU = 0;
const DefaultPayloadTypePCMA = 8;
const DefaultPayloadTypeG722 = 9;
const DefaultPayloadTypeOpus = 111;
const DefaultPayloadTypeVP8 = 96;
const DefaultPayloadTypeVP9 = 98;
const DefaultPayloadTypeH264 = 102;

export default class Client extends EventEmitter {

    constructor() {
        super();
        this._port = 8443;
        this._uid = uuidv4();
        this._pcs = new Map();
        this._streams = new Map();
    }

    get uid() {
        return this._uid;
    }

    init() {
        this._url = this._getProtooUrl(this._uid);

        let transport = new protooClient.WebSocketTransport(this._url);
        this._protoo = new protooClient.Peer(transport);

        this._protoo.on('open', () => {
            console.log('Peer "open" event');
            this.emit('transport-open');
        });

        this._protoo.on('disconnected', () => {
            console.log('Peer "disconnected" event');
            this.emit('transport-failed');
        });

        this._protoo.on('close', () => {
            console.log('Peer "close" event');
            this.emit('transport-closed');
        });

        this._protoo.on('request', this._handleRequest.bind(this));
        this._protoo.on('notification', this._handleNotification.bind(this));
    }

    async join(channelId) {
        this._rid = channelId;
        try {
            let data = await this._protoo.request('join', { 'rid': this._rid, 'id': this._uid });
            console.log('join success: result => ' + JSON.stringify(data));
        } catch (error) {
            console.log('join reject: error =>' + error);
        }
    }

    async leave() {
        try {
            let data = await this._protoo.request('leave', { 'rid': this._rid, 'id': this._uid });
            console.log('leave success: result => ' + JSON.stringify(data));
        } catch (error) {
            console.log('leave reject: error =>' + error);
        }
    }

    async publish(options = { audio: true, video: true, screen: false, codec: 'vp8' }) {
        console.log('publish options => %o', options);
        var promise = new Promise(async (resolve, reject) => {
            try {
                if (this._pcs[this._uid] != null) {
                    throw 'already in publish, abort!';
                }
                let stream = new Stream(this._uid);
                await stream.init({ audio: options.audio, video: options.video, screen: options.screen });
                let pc = await this._createSender(this._uid, stream.stream, options.codec);

                pc.onicecandidate = async (e) => {
                    if (!pc.sendOffer) {
                        var offer = pc.localDescription;
                        console.log('Send offer sdp => ' + offer.sdp);
                        pc.sendOffer = true
                        let answer = await this._protoo.request('publish', { jsep: offer });
                        await pc.setRemoteDescription(answer.jsep);
                        console.log('publish success => ' + JSON.stringify(answer));
                        resolve(stream);
                    }
                }
            } catch (error) {
                throw error;
                console.log('publish request error  => ' + error);
                reject(error);
            }
        });
        return promise;
    }

    async unpublish() {
        console.log('unpublish uid => %s', this._uid);
        try {
            let data = await this._protoo.request('unpublish', { 'rid': this._rid });
            console.log('unpublish success: result => ' + JSON.stringify(data));
            this._removePC(this._uid);
        } catch (error) {
            console.log('unpublish reject: error =>' + error);
        }
    }

    async subscribe(pid) {
        console.log('subscribe pid => %s', pid);
        var promise = new Promise(async (resolve, reject) => {
            try {
                let pc = await this._createReceiver(pid);
                pc.onaddstream = (e) => {
                    var stream = e.stream;
                    console.log('Stream::pc::onaddstream', stream.id);
                    resolve(new Stream(pid, stream));
                }
                pc.onremovestream = (e) => {
                    var stream = e.stream;
                    console.log('Stream::pc::onremovestream', stream.id);
                }
                pc.onicecandidate = async (e) => {
                    if (!pc.sendOffer) {
                        var jsep = pc.localDescription;
                        console.log('Send offer sdp => ' + jsep.sdp);
                        pc.sendOffer = true
                        let answer = await this._protoo.request('subscribe', { pid, jsep });
                        console.log('subscribe success => answer(' + pid + ') sdp => ' + answer.jsep.sdp);
                        await pc.setRemoteDescription(answer.jsep);
                    }
                }
            } catch (error) {
                console.log('subscribe request error  => ' + error);
                reject(error);
            }
        });
        return promise;
    }

    async unsubscribe(pid) {
        console.log('unsubscribe pid => %s', pid);
        try {
            let data = await this._protoo.request('unsubscribe', { pid });
            console.log('unsubscribe success: result => ' + JSON.stringify(data));
            this._removePC(pid);
        } catch (error) {
            console.log('unsubscribe reject: error =>' + error);
        }
    }

    close() {
        this._protoo.close();
    }

    _payloadModify(desc, codec) {

        if (codec === undefined)
            return desc;

        const session = sdpTransform.parse(desc.sdp);
        console.log('SDP object => %o', session);
        var videoIdx = 1;
        /*
         * DefaultPayloadTypePCMU = 0
         * DefaultPayloadTypePCMA = 8
         * DefaultPayloadTypeG722 = 9
         * DefaultPayloadTypeOpus = 111
         * DefaultPayloadTypeVP8  = 96
         * DefaultPayloadTypeVP9  = 98
         * DefaultPayloadTypeH264 = 102
        */
        let payload;
        let codeName = '';
        if (codec.toLowerCase() === 'vp8') {
            /*Add VP8 and RTX only.*/
            payload = DefaultPayloadTypeVP8;
            codeName = "VP8";
        } else if (codec.toLowerCase() === 'vp9') {
            payload = DefaultPayloadTypeVP9;
            codeName = "VP9";
        } else if (codec.toLowerCase() === 'h264') {
            payload = DefaultPayloadTypeH264;
            codeName = "H264";
        } else {
            return desc;
        }

        var rtp = [
            { "payload": payload, "codec": codeName, "rate": 90000, "encoding": null },
            { "payload": 97, "codec": "rtx", "rate": 90000, "encoding": null }
        ];

        session['media'][videoIdx]["payloads"] = payload + " 97";
        session['media'][videoIdx]["rtp"] = rtp;

        var fmtp = [
            { "payload": 97, "config": "apt=" + payload }
        ];

        session['media'][videoIdx]["fmtp"] = fmtp;

        var rtcpFB = [
            { "payload": payload, "type": "transport-cc", "subtype": null },
            { "payload": payload, "type": "ccm", "subtype": "fir" },
            { "payload": payload, "type": "nack", "subtype": null },
            { "payload": payload, "type": "nack", "subtype": "pli" }
        ];
        session['media'][videoIdx]["rtcpFb"] = rtcpFB;

        let tmp = desc;
        tmp.sdp = sdpTransform.write(session);
        return tmp;
    }

    async _createSender(uid, stream, codec) {
        console.log('create sender => %s', uid);
        let pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] });
        pc.sendOffer = false;
        pc.addStream(stream);
        let offer = await
            pc.createOffer({ offerToReceiveVideo: false, offerToReceiveAudio: false });
        let desc = this._payloadModify(offer, codec);
        pc.setLocalDescription(desc);
        this._pcs[uid] = pc;
        return pc;
    }

    async _createReceiver(uid) {
        console.log('create receiver => %s', uid);
        let pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] });
        pc.sendOffer = false;
        pc.addTransceiver('audio', { 'direction': 'recvonly' });
        pc.addTransceiver('video', { 'direction': 'recvonly' });
        let desc = await pc.createOffer();
        pc.setLocalDescription(desc);
        this._pcs[uid] = pc;
        return pc;
    }

    _removePC(uid) {
        let pc = this._pcs[uid];
        if (pc) {
            console.log('remove pc => %s', uid);
            pc.close();
            delete this._pcs[uid];
        }
    }

    _getProtooUrl(pid) {
        const hostname = window.location.hostname;
        let url = `wss://${hostname}:${this._port}/ws?peer=${pid}`;
        return url;
    }

    _handleRequest(request, accept, reject) {
        console.log('Handle request from server: [method:%s, data:%o]', request.method, request.data);
    }

    _handleNotification(notification) {
        console.log('Handle notification from server: [method:%s, data:%o]', notification.method, notification.data);
        switch (notification.method) {
            case 'peer-join':
                {
                    let pid = notification.data.id;
                    let rid = notification.data.rid;
                    console.log('peer-join peer id => ' + pid);
                    this.emit('peer-join', pid, rid);
                    break;
                }
            case 'peer-leave':
                {
                    let pid = notification.data.id;
                    let rid = notification.data.rid;
                    console.log('peer-leave peer id => ' + pid);
                    this.emit('peer-leave', pid, rid);
                    this._removePC(pid);
                    break;
                }
            case 'stream-add':
                {
                    let pid = notification.data.pid;
                    let rid = notification.data.rid;
                    console.log('stream-add peer id => ' + pid);
                    this.emit('stream-add', pid, rid);
                    break;
                }
            case 'stream-remove':
                {
                    let pid = notification.data.pid;
                    let rid = notification.data.rid;
                    console.log('stream-remove peer id => ' + pid);
                    this.emit('stream-remove', pid, rid);
                    this._removePC(pid);
                    break;
                }
        }
    }
}
