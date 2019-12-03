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
        <div
          style={{
            position: "absolute",
            top: 64,
            left: 0,
            right: 0,
            bottom: 0
          }}
        >
          <video
            ref={id}
            id={id}
            autoPlay
            playsInline
            muted={false}
            style={{
              width: "100%",
              height: "100%",
              objectFit: "cover"
            }}
          />
        </div>
        <div style={{ position: "absolute", left: 0, right: 0 }}>
          <div>
            <a style={{ position: "absolute", left: 8, top: 8 }}>{id}</a>
          </div>
        </div>
      </div>
    );
  };
}

export default MainVideoView;
