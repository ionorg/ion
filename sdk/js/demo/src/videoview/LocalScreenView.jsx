import React from "react";
import { Button } from "antd";
import { setTimeout } from "timers";

class LocalScreenView extends React.Component {

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

  render = () => {
    const { id } = this.props;
    return (
      <div className="local-video-border">
        <div className="local-video-layout">
          <div>
            <a className="local-video-name">Screen Sharing</a>
          </div>
        </div>

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
        </div>
      </div>
    );
  };
}

export default LocalScreenView;
