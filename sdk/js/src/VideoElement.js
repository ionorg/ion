
class VideoElement {
    constructor() {
    }

    play(options = { id, stream, elementId, remote: false }) {
        let video = document.createElement('video');
        video.autoplay = true;
        video.playsinline = true;
        video.controls = true;
        video.muted = !options.remote;
        video.srcObject = options.stream;
        video.id = `stream${options.id}`;
        let parentElement = document.getElementById(options.elementId);
        parentElement.appendChild(video);
        this.parentElement = parentElement;
        this._video = video;
    }

    stop() {
        this._video.stop();
        this.parentElement.removeChild(this._video);
    }
}

export default VideoElement;
