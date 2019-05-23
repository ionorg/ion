/* global  */

class Player {
    constructor (opt) {
        this._create(opt);
    }

    _create ({id, stream, parent}) {
        let video = document.createElement('video');
        video.class = 'player';
        video.style = 'width: 320px; height: 240px;';
        video.autoplay = true;
        video.playsinline = true;
        video.controls = true;
        video.muted = true;
        video.srcObject = stream;
        video.id = `stream${id}`;
        this.video = video;
        let parentElement = document.getElementById(parent);
        parentElement.appendChild(video);
        this.parentElement = parentElement;
    }

    destroy () {
        this.video.pause();
        this.parentElement.removeChild(this.video);
    }
}

export default Player;
