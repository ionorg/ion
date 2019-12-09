import React from "react";
import { setTimeout } from "timers";

class LocalVideoView extends React.Component {
  state = {
    type: "video",
    enabled: true
  };

  componentDidMount = () => {
    const { handleStreamEnabled } = this.props;
    const { type, enabled } = this.state;
    setTimeout(() => {
      handleStreamEnabled(type, enabled);
    }, 500);
  };

  componentWillUnmount = () => {
    let stream = this.video.srcObject;
    if (stream !== null) {
      let tracks = stream.getTracks();
      for (let i = 0, len = tracks.length; i < len; i++) {
        tracks[i].stop();
      }
    }
    this.video.srcObject = null;
  };

  set stream(stream) {
    if (stream != null) {
      this.video.srcObject = stream.stream;
    } else {
      stream = this.video.srcObject;
      if (stream !== null) {
        let tracks = stream.getTracks();
        for (let i = 0, len = tracks.length; i < len; i++) {
          tracks[i].stop();
        }
      }
      this.video.srcObject = null;
    }
  }

  _handleStreamEnabled = (type, enabled) => {
    const { handleStreamEnabled } = this.props;
    this.setState({ type, enabled });
    handleStreamEnabled(type, enabled);
  };

  render = () => {
    const { id } = this.props;
    return (
      <div className="local-video-layout">
        <div className="local-video-container">
          <video
            ref={ref => {
              this.video = ref;
            }}
            id={id}
            autoPlay
            playsInline
            muted={true}
            className="local-video-size"
          />
          <a className="local-video-name">Local</a>
        </div>  
      </div>
    );
  };
}

export default LocalVideoView;