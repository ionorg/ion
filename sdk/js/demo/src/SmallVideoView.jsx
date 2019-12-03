import React from "react";

class SmallVideoView extends React.Component {
  state = {
    payling: false
  };

  componentDidMount = () => {
    const { id, stream } = this.props;
    let video = this.refs[id];
    video.srcObject = stream.stream;
  };

  _onChangeVideoPosition = () => {
    this.props.onClick({id:this.props.id,index:this.props.index});
  }

  render = () => {
    const { id } = this.props;
    return (
      <div
        onClick={this._onChangeVideoPosition}
        style={{
          borderWidth: "0.5px",
          borderStyle: "solid",
          borderColor: "#ffffff",
          overflow: "hidden",
          borderRadius: "2px",
          backgroundColor: 'rgb(0, 21, 42)'
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
          <div>
            <a style={{ position: "absolute", left: -10, top: -16 }}>{id}</a>
          </div>
        </div>
      </div>
    );
  };
}

export default SmallVideoView;
