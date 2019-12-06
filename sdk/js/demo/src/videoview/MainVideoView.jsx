import React from "react";

class MainVideoView extends React.Component {
  componentDidMount = () => {
    const { id, stream } = this.props;
    let video = this.refs[id];
    video.srcObject = stream.stream;
  };

  render = () => {
    const { id } = this.props;
    return (
      <div>
        <div className="main-video-layout">
          <video
            ref={id}
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
