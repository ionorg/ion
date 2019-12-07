import React from "react";

class MainVideoView extends React.Component {
  componentDidMount = () => {
    const { id, stream } = this.props;
    let video = this.video;
    video.srcObject = stream.stream;
  };

  componentWillUnmount = () => {
    const {id } = this.props;
    let video = this.video;
    let stream = video.srcObject;
    if (stream !== null) {
      let tracks = stream.getTracks();
      for (let i = 0, len = tracks.length; i < len; i++) {
        tracks[i].stop();
      }
    }
    video.srcObject = null;
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
            ref={ref => {
              this.video = ref;
            }}
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
