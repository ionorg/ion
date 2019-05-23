/* global */
// eslint-disable-next-line
//
import { EventEmitter } from 'events';
import protooClient from 'protoo-client';
const uuidv4 = require('uuid/v4');

const protooPort = 8443;

class Room extends EventEmitter {

    constructor() {
        super()
        this.uid = uuidv4();
        this.url = this.getProtooUrl(this.uid);
        let transport = new protooClient.WebSocketTransport(this.url);
        // protoo-client Peer instance.
        this._protoo = new protooClient.Peer(transport);

        this._protoo.on('open', () => {
            console.log('Peer "open" event');
            this.emit('onRoomConnect');
        });

        this._protoo.on('disconnected', () => {
            console.log('Peer "disconnected" event');
            this.emit('onRoomDisconnect');
        });

        this._protoo.on('close', () => {
            console.log('Peer "close" event');
            this.emit('onRoomDisconnect');
        });

        this._protoo.on('request', this._handleRequest.bind(this));
        this._protoo.on('notification', this._handleNotification.bind(this))
    }

    getProtooUrl(peerId)
    {
        const hostname = window.location.hostname;
        let url = `wss://${hostname}:${protooPort}/ws?peer=${peerId}`;
        return url;
    }

    async join(roomId) {
        this.rid = roomId;
        try{
            let data = await this._protoo.request('join', {'rid': this.rid});
            console.log('join success: result => ' + JSON.stringify(data));
        }catch(error) {
            console.log('join reject: error =>' + error);
        }
    }

    async publish(offer, pubid){
        try {
            let answer = await this._protoo.request('publish',{jsep: offer, pubid});
            console.log('publish success => ' + JSON.stringify(answer));
            return answer;
        }catch(error) {
            throw error;
        }
    }

    async subscribe(offer, pubid){
        try {
            let answer = await this._protoo.request('subscribe',{jsep: offer, pubid});
            console.log('subscribe success => ' + JSON.stringify(answer));
            return answer;
        }catch(error) {
            throw error;
        }
    }

    close() {
        this._protoo.close();
    }

    leave() {
        this._protoo.request('leave',{'rid': this.rid});
    }

    _handleRequest(request, accept, reject) {
        console.log('Handle request from server: [method:%s, data:%o]', request.method, request.data);
    }

    _handleNotification (notification) {
        console.log('Handle notification from server: [method:%s, data:%o]', notification.method, notification.data);

        switch(notification.method){
            case 'onPublish':
                {
                    let pubid = notification.data.pubid;
                    console.log('Got publish from => ' + pubid);
                    this.emit('onRtcCreateRecver', pubid);
                    break;
                }
                case 'onUnpublish':
                {
                    let pubid = notification.data.pubid;
                    console.log('[' + pubid + ']  => leave !!!!');
                    this.emit('onRtcLeaveRecver', pubid);
                    break;
                }
        }
    }
}

export default Room