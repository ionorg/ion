import React from "react";

class MainVideoView extends React.Component {
  componentDidMount = () => {
    const { stream } = this.props;
    this.video.srcObject = stream.stream;
  };

  render = () => {
    const { id } = this.props;
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
        <a className="main-video-id-a">{id}</a>
      </div>
    );
  };
}

export default MainVideoView;