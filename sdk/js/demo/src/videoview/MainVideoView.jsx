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
      <div className="main-video-layout">
          <video
            ref={id}
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
