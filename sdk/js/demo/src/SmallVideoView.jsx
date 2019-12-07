import React from "react";

class SmallVideoView extends React.Component {
  state = {
    payling: false
  };

  componentDidMount = () => {
    const { id, stream } = this.props;
    let video = this.video;
    video.srcObject = stream.stream;
  };

  componentWillUnmount = () => {
    const { id } = this.props;
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
      <div
        style={{
          borderWidth: "0.5px",
          borderStyle: "solid",
          borderColor: "#ffffff",
          overflow: "hidden",
          borderRadius: "2px",
          backgroundColor: "rgb(0, 21, 42)"
        }}
      >
        <div
          style={{
            width: 220,
            height: 140,
            zIndex: 0
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
      </div>
    );
  };
}

export default SmallVideoView;
