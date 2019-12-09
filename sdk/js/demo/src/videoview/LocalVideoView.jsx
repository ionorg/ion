import React from "react";

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
    return (
      <div className="local-video-layout">
        <div className="local-video-container">
          <video
            ref={id}
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
