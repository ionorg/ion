/* global  */

class Player {
    constructor (opt) {
        this.create(opt)
    }

    create (opt) {
        this.video = document.createElement('video')
        this.video.class = 'player'
        this.video.style = 'width: 320px; height: 240px; '
        this.video.autoplay = true
        this.video.playsinline = true
        this.video.controls = true
        this.video.muted = true
        this.video.srcObject = opt.stream
        this.video.id = `stream${opt.id}`
        document.body.appendChild(this.video)
    }
    addTrack (track) {
        this.video.addTrack(track)
    }
    destroy () {
        this.video.pause()
        document.body.removeChild(this.video)
    }
}

export default Player
