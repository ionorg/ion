import React from "react";

class MainVideoView extends React.Component {
  componentDidMount = () => {
    const { stream } = this.props;
    this.video.srcObject = stream.stream;
  };

  componentWillUnmount = () => {
    this.video.srcObject = null;
  }

  render = () => {
    const { id, stream } = this.props;
    return (
      <div className="main-video-layout">
        <video
          ref={ref => {
            this.video = ref;
          }}
          id={id}
          autoPlay
          playsInline
          muted={false}
          className="main-video-size"
        />
        <div className="main-video-name">
          <a className="main-video-name-a">{stream.info.name}</a>
        </div>
      </div>
    );
  };
}

export default MainVideoView;
