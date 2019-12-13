import React from "react";

class SmallVideoView extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      clientWidth:document.body.offsetWidth,
      clientHeight:document.body.offsetHeight,
    }
  }

  componentDidMount = () => {
    const { stream } = this.props;
    this.video.srcObject = stream.stream;

    
  };

  _handleClick = () => {
    let { id, index } = this.props;
    this.props.onClick({ id, index });
  };

  render = () => {
    const { id, stream } = this.props;

    let videoCount = this.props.videoCount;
    let leftWidth = this.props.collapsed ? 0 : 320;

    let clientWidth = this.state.clientWidth;
    let videoContainerW = clientWidth - leftWidth;
    let startX = (videoContainerW - (videoCount-1) * 220)/2;

    const small = {
      position: "absolute",
      width: "220px",
      height: "140px",
      left: startX + (this.props.index - 1) * 220 + "px",
      bottom: "30px",
      borderWidth: "0.5px",
      borderStyle: "solid",
      borderColor: "#ffffff",
      overflow: "hidden",
      borderRadius: "2px",
      backgroundColor: "#323232"
    };

    return (
      <div onClick={this._handleClick} style={small}>
        <video
          ref={ref => {
            this.video = ref;
          }}
          id={id}
          autoPlay
          playsInline
          muted={false}
          className="small-video-size"
        />
        <a className="small-video-id-a">{stream.info.name}</a>
      </div>
    );
  };
}

export default SmallVideoView;
