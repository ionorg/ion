import React from 'react';
import { Modal, Button, Select,Tooltip } from 'antd';
import SoundMeter from './soundmeter';
import PropTypes from 'prop-types';

const Option = Select.Option;

const closeMediaStream = function (stream) {
    if (!stream) {
        return;
    }
    if (MediaStreamTrack && MediaStreamTrack.prototype && MediaStreamTrack.prototype.stop) {
        var tracks, i, len;

        if (stream.getTracks) {
            tracks = stream.getTracks();
            for (i = 0, len = tracks.length; i < len; i += 1) {
                tracks[i].stop();
            }
        } else {
            tracks = stream.getAudioTracks();
            for (i = 0, len = tracks.length; i < len; i += 1) {
                tracks[i].stop();
            }

            tracks = stream.getVideoTracks();
            for (i = 0, len = tracks.length; i < len; i += 1) {
                tracks[i].stop();
            }
        }
        // Deprecated by the spec, but still in use.
    } else if (typeof stream.stop === 'function') {
        console.log('closeMediaStream() | calling stop() on the MediaStream');
        stream.stop();
    }
}

// Attach a media stream to an element.
const attachMediaStream = function (element, stream) {
    element.srcObject = stream;
};

export default class MediaSettings extends React.Component {

    constructor(props) {
        super(props)
        let settings = props.settings;
        this.state = {
            visible: false,
            videoDevices: [],
            audioDevices: [],
            audioOutputDevices: [],
            resolution: settings.resolution,
            bandwidth: settings.bandwidth,
            selectedAudioDevice: settings.selectedAudioDevice,
            selectedVideoDevice: settings.selectedVideoDevice,
            codec: settings.codec,
        }

        try {
            window.AudioContext = window.AudioContext || window.webkitAudioContext;
            window.audioContext = new AudioContext();
        } catch (e) {
            console.log('Web Audio API not supported.');
        }
    }

    updateInputDevices = () => {
        return new Promise((pResolve, pReject) => {
            let videoDevices = [];
            let audioDevices = [];
            let audioOutputDevices = [];
            navigator.mediaDevices.enumerateDevices()
            .then((devices) => {
                for (let device of devices) {
                    if (device.kind === 'videoinput') {
                        videoDevices.push(device);
                    } else if (device.kind === 'audioinput') {
                        audioDevices.push(device);
                    }else if (device.kind === 'audiooutput'){
                        audioOutputDevices.push(device);
                    }
                }
            }).then(() => {
                let data = { videoDevices, audioDevices, audioOutputDevices};
                pResolve(data);
            });
        });
    }

    componentDidMount() {
        this.updateInputDevices().then((data) => {

            if (this.state.selectedAudioDevice === "" && data.audioDevices.length > 0) {
                this.state.selectedAudioDevice = data.audioDevices[0].deviceId;
            }
            if (this.state.selectedVideoDevice === "" && data.videoDevices.length > 0) {
                this.state.selectedVideoDevice = data.videoDevices[0].deviceId;
            }

            this.setState({
                videoDevices: data.videoDevices,
                audioDevices: data.audioDevices,
                audioOutputDevices: data.audioOutputDevices,
             })

            this.state.audioDevices.map((device, index) => {
                if (this.state.selectedAudioDevice == device.deviceId) {
                    console.log("Selected audioDevice::" + JSON.stringify(device));
                }
            });
            this.state.videoDevices.map((device, index) => {
                if (this.state.selectedVideoDevice == device.deviceId) {
                    console.log("Selected videoDevice::" + JSON.stringify(device));
                }
            });
        });

    }

    soundMeterProcess = () => {
        var val = (window.soundMeter.instant.toFixed(2) * 348) + 1;
        this.setState({ audioLevel: val });
        if (this.state.visible)
            setTimeout(this.soundMeterProcess, 100);
    }

    startPreview = () => {
        if (window.stream) {
            closeMediaStream(window.stream);
        }
        let videoElement = this.refs['previewVideo'];
        let audioSource = this.state.selectedAudioDevice;
        let videoSource = this.state.selectedVideoDevice;
        this.soundMeter = window.soundMeter = new SoundMeter(window.audioContext);
        let soundMeterProcess = this.soundMeterProcess;
        let constraints = {
            audio: { deviceId: audioSource ? { exact: audioSource } : undefined },
            video: { deviceId: videoSource ? { exact: videoSource } : undefined }
        };
        navigator.mediaDevices.getUserMedia(constraints)
            .then(function (stream) {
                window.stream = stream; // make stream available to console
                //videoElement.srcObject = stream;
                attachMediaStream(videoElement, stream);
                soundMeter.connectToSource(stream);
                setTimeout(soundMeterProcess, 100);
                // Refresh button list in case labels have become available
                return navigator.mediaDevices.enumerateDevices();
            })
            .then((devces) => { })
            .catch((erro) => { });
    }

    stopPreview = () => {
        if (window.stream) {
            closeMediaStream(window.stream);
        }
    }

    componentWillUnmount() {

    }

    showModal = () => {
        this.setState({
            visible: true,
        });
        setTimeout(this.startPreview, 100);
    }

    handleOk = (e) => {
        console.log(e);
        this.setState({
            visible: false,
        });
        this.stopPreview();
        if(this.props.onMediaSettingsChanged !== undefined) {
            this.props.onMediaSettingsChanged(
                this.state.selectedAudioDevice,
                this.state.selectedVideoDevice,
                this.state.resolution,
                this.state.bandwidth,
                this.state.codec);
        }
    }

    handleCancel = (e) => {
        this.setState({
            visible: false,
        });
        this.stopPreview();
    }

    handleAudioDeviceChange = (e) => {
        this.setState({ selectedAudioDevice: e });
        setTimeout(this.startPreview, 100);
    }

    handleVideoDeviceChange = (e) => {
        this.setState({ selectedVideoDevice: e });
        setTimeout(this.startPreview, 100);
    }

    handleResolutionChange = (e) => {
        this.setState({ resolution: e });
    }

    handleVideoCodeChange = (e) => {
        this.setState({ codec: e });
    }

    handleBandWidthChange = (e) => {
        this.setState({ bandwidth: e });
    }

    render() {
        return (
            <div>
                {
                    <Tooltip title='System setup'>
                        <Button shape="circle" icon="setting" ghost onClick={this.showModal}/>
                    </Tooltip>
                }
                <Modal
                    title='Settings'
                    visible={this.state.visible}
                    onOk={this.handleOk}
                    onCancel={this.handleCancel}
                    okText='Ok'
                    cancelText='Cancel'>
                    <div className="settings-item">
                        <span className="settings-item-left">Micphone</span>
                        <div className="settings-item-right">
                            <Select value={this.state.selectedAudioDevice} style={{ width: 350 }} onChange={this.handleAudioDeviceChange}>
                                {
                                    this.state.audioDevices.map((device, index) => {
                                        return (<Option value={device.deviceId} key={device.deviceId}>{device.label}</Option>);
                                    })
                                }
                            </Select>
                            <div ref="progressbar" style={{
                                width: this.state.audioLevel + 'px',
                                height: '10px',
                                backgroundColor: '#8dc63f',
                                marginTop: '20px',
                            }}>
                            </div>
                        </div>
                    </div>
                    <div className="settings-item">
                        <span className="settings-item-left">Camera</span>
                        <div className="settings-item-right">
                            <Select value={this.state.selectedVideoDevice} style={{ width: 350 }} onChange={this.handleVideoDeviceChange}>
                                {
                                    this.state.videoDevices.map((device, index) => {
                                        return (<Option value={device.deviceId} key={device.deviceId}>{device.label}</Option>);
                                    })
                                }
                            </Select>
                            <div className="settings-video-container">
                                <video id='previewVideo' ref='previewVideo' autoPlay playsInline muted={true} style={{ width: '100%', height: '100%', objectFit: 'contain' }}></video>
                            </div>

                        </div>
                    </div>
                    <div className="settings-item">
                        <span className="settings-item-left">Quality</span>
                        <div className="settings-item-right">
                            <Select style={{ width: 350 }} value={this.state.resolution} onChange={this.handleResolutionChange}>
                                <Option value="qvga">QVGA(320x180)</Option>
                                <Option value="vga">VGA(640x360)</Option>
                                <Option value="shd">SHD(960x540)</Option>
                                <Option value="hd">HD(1280x720)</Option>
                            </Select>
                        </div>
                    </div>
                    <div className="settings-item">
                        <span className="settings-item-left">VideoCode</span>
                        <div className="settings-item-right">
                            <Select style={{ width: 350 }} value={this.state.codec} onChange={this.handleVideoCodeChange}>
                                <Option value="h264">H264</Option>
                                <Option value="vp8">VP8</Option>
                                <Option value="vp9">VP9</Option>
                            </Select>
                        </div>
                    </div>
                    <div className="settings-item">
                        <span className="settings-item-left">Bandwidth</span>
                        <div className="settings-item-right">
                            <Select style={{ width: 350 }} value={this.state.bandwidth} onChange={this.handleBandWidthChange}>
                                <Option value="256">Low(256kbps)</Option>
                                <Option value="512">Medium(512kbps)</Option>
                                <Option value="1024">High(1Mbps)</Option>
                                <Option value="4096">Lan(4Mbps)</Option>
                            </Select>
                        </div>
                    </div>
                </Modal>
            </div>
        );
    }
}


MediaSettings.propTypes = {
    onMediaSettingsChanged: PropTypes.func
}