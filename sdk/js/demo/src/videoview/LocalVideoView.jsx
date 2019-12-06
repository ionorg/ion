import React from "react";
import { Button } from "antd";

class LocalVideoView extends React.Component {
  state = {
    type: "video",
    enabled: true
  };

  componentDidMount = () => {
    const { handleStreamEnabled } = this.props;
    const { type, enabled } = this.state;
    handleStreamEnabled(type, enabled);
  };

  componentWillUnmount = () => {
    const {id } = this.props;
    let video = this.refs[id];
    let stream = video.srcObject;
    if (stream !== null) {
      let tracks = stream.getTracks();
      for (let i = 0, len = tracks.length; i < len; i++) {
        tracks[i].stop();
      }
    }
    video.srcObject = null;
  };

  set stream(stream) {
    const { id } = this.props;
    let video = this.refs[id];
    if (stream != null) {
      let video = this.refs[id];
      video.srcObject = stream.stream;
    } else {
      stream = video.srcObject;
      if (stream !== null) {
        let tracks = stream.getTracks();
        for (let i = 0, len = tracks.length; i < len; i++) {
          tracks[i].stop();
        }
      }
      video.srcObject = null;
    }
  }

  _handleStreamEnabled = (type, enabled) => {
    const { handleStreamEnabled } = this.props;
    this.setState({ type, enabled });
    handleStreamEnabled(type, enabled);
  };

  render = () => {
    const { id } = this.props;
    const { enabled, type } = this.state;
    return (
      <div className="local-video-border">
        <div className="local-video-layout">
          <div>
            <a className="local-video-name">
              Local
            </a>
            <div className="local-video-tool">
              <div style={{ display: "inline" }}>
                <Button
                  shape="circle"
                  icon="audio"
                  disabled={enabled && type != "audio"}
                  onClick={() => this._handleStreamEnabled("audio", !enabled)}
                />
                <Button
                  shape="circle"
                  icon="video-camera"
                  disabled={enabled && type != "video"}
                  onClick={() => this._handleStreamEnabled("video", !enabled)}
                />
                <Button
                  shape="circle"
                  icon="desktop"
                  disabled={enabled && type != "screen"}
                  onClick={() => this._handleStreamEnabled("screen", !enabled)}
                />
              </div>
            </div>
          </div>
        </div>

        <div className="local-video-container">
          <video
            ref={id}
            id={id}
            autoPlay
            playsInline
            muted={true}
            className="local-video-size"
          />
        </div>
      </div>
    );
  };
}

export default LocalVideoView;
