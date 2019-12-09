import React from "react";

class MainVideoView extends React.Component {
  componentDidMount = () => {
    const { stream } = this.props;
    this.video.srcObject = stream.stream;
  };

  render = () => {
    const { id } = this.props;
    return (
      <div>
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
        </div>
        <div className="main-video-id-container">
          <div>
            <a className="main-video-id-a">{id}</a>
          </div>
        </div>
      </div>
    );
  };
}

export default MainVideoView;
