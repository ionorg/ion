/* global */
// eslint-disable-next-line
//
import Centrifuge from 'centrifuge'
import { EventEmitter } from 'events';

class Signal extends EventEmitter {
    constructor (opt) {
        super()
        this.signal = new Centrifuge(opt.url)
        this.signal.setToken(opt.token)
        this.roomID = opt.roomid
    }

    connect () {

        this.signal.on('connect', (context) => {
            this.emit('connect', context)
            console.log('Signal.connect', context)
        })

        this.signal.on('disconnect', (context) => {
            this.emit('disconnect', context);
            console.log('Signal.disconnect', context)
        })

        try {
            this.signal.connect()
        } catch (e) {
            console.log('Signal.connect fail')
            this.signal.connect()
        }
    }

    disconnect () {
        this.signal.disconnect()
    }

    broadcast (msg) {
        this.signal.publish(this.roomID, msg)
    }

    publish (msg) {
        this.signal.publish(this.clientID, msg)
    }

    subscribe (channel, cb) {
        this.signal.subscribe(channel, cb)
        this.clientID = channel
    }

}
export default Signal
