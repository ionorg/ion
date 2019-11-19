import { EventEmitter } from 'events';
import protooClient from 'protoo-client';
import uuidv4 from 'uuid/v4';

import Streeam from './Stream';

const ices = 'stun:stun.stunprotocol.org:3478';

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

    async publish(options = { audio: true, video: true, screen: false }) {
        var promise = new Promise(async (resolve, reject) => {
            try {
                if (this._pcs[this._uid] != null) {
                    throw 'already in publish, abort!';
                }
                let stream = new Stream(this._uid);
                await stream.init({ audio: options.audio, video: options.video, screen: options.screen });
                let pc = await this._createSender(this._uid, stream.stream);

                pc.onicecandidate = async (e) => {
                    if (!pc.sendOffer) {
                        var offer = pc.localDescription;
                        console.log('Send offer sdp => ' + offer.sdp);
                        pc.sendOffer = true

                        let answer = await this._protoo.request('publish', { jsep: offer });
                        await pc.setRemoteDescription(answer.jsep);
                        resolve(stream);
                        console.log('publish success => ' + JSON.stringify(answer));
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
        try {
            let data = await this._protoo.request('unpublish', { 'rid': this._rid });
            console.log('unpublish success: result => ' + JSON.stringify(data));
            let pc = this._pcs[this._uid];
            if (pc) {
                pc.close();
                delete this._pcs[this._uid];
            }
        } catch (error) {
            console.log('unpublish reject: error =>' + error);
        }
    }

    async subscribe(pid) {
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
                        await pc.setRemoteDescription(answer.jsep);
                        console.log('subscribe success => answer(' + pid + ') sdp => ' + answer.jsep.sdp);
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
        try {
            let data = await this._protoo.request('unsubscribe', { pid });
            console.log('unsubscribe success: result => ' + JSON.stringify(data));
            let pc = this._pcs[pid];
            if (pc) {
                pc.close();
                delete this._pcs[pid];
            }
        } catch (error) {
            console.log('unsubscribe reject: error =>' + error);
        }
    }

    close() {
        this._protoo.close();
    }

    async _createSender(uid, stream) {
        let pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] });
        pc.sendOffer = false;
        pc.addStream(stream);
        let desc = await
            pc.createOffer({ offerToReceiveVideo: false, offerToReceiveAudio: false });
        pc.setLocalDescription(desc);
        this._pcs[uid] = pc;
        return pc;
    }

    async _createReceiver(uid) {
        let pc = new RTCPeerConnection({ iceServers: [{ urls: ices }] });
        pc.sendOffer = false;
        pc.addTransceiver('audio', { 'direction': 'recvonly' });
        pc.addTransceiver('video', { 'direction': 'recvonly' });
        let desc = await pc.createOffer();
        pc.setLocalDescription(desc);
        this._pcs[uid] = pc;
        return pc;
    }

    _getProtooUrl(peerId) {
        const hostname = window.location.hostname;
        let url = `wss://${hostname}:${this._port}/ws?peer=${peerId}`;
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
                    let peerId = notification.data.id;
                    let rid = notification.data.rid;
                    console.log('peer-join peer id => ' + peerId);
                    this.emit('peer-join', peerId, rid);
                    break;
                }
            case 'peer-leave':
                {
                    let peerId = notification.data.id;
                    let rid = notification.data.rid;
                    console.log('peer-leave peer id => ' + peerId);
                    this.emit('peer-leave', peerId, rid);
                    break;
                }
            case 'stream-add':
                {
                    let peerId = notification.data.pid;
                    let rid = notification.data.rid;
                    console.log('stream-add peer id => ' + peerId);
                    this.emit('stream-add', peerId, rid);
                    break;
                }
            case 'stream-remove':
                {
                    let peerId = notification.data.pid;
                    let rid = notification.data.rid;
                    console.log('stream-remove peer id => ' + peerId);
                    this.emit('stream-remove', peerId, rid);
                    break;
                }
            case 'stream-subscribed':
                {
                    let peerId = notification.data.pid;
                    let rid = notification.data.rid;
                    console.log('stream-subscribed peer id => ' + peerId);
                    this.emit('stream-subscribed', peerId, rid);
                    break;
                }
        }
    }
}
