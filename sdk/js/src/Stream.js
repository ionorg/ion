import { EventEmitter } from 'events';
import VideoElement from './VideoElement';

export default class Stream extends EventEmitter {

    constructor(uid, stream = null) {
        super();
        this._uid = uid;
        this._stream = stream;
        this._videoElement = new VideoElement();
    }

    async init(sender = false, options = { audio: true, video: true, screen: false }) {
        if (sender) {
            if (options.screen) {
                this._stream = await navigator.mediaDevices.getDisplayMedia({ video: true });
            } else {
                this._stream = await navigator.mediaDevices.getUserMedia({ audio: options.audio, video: options.video });
            }
        }
    }

    get uid() { return this._uid; }

    get stream() { return this._stream };

    render(elementId) {
        this._videoElement.play({ id: this._uid, stream: this._stream, elementId });
    }

    async stop() {
        this._videoElement.stop();
    }
}