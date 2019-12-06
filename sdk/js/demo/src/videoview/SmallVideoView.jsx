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
      <div onClick={this._onChangeVideoPosition} className="small-video-border">
        <div className="small-video-container">
          <video
            ref={id}
            id={id}
            autoPlay
            playsInline
            muted={false}
            className="small-video-size"
          />
          <div>
            <a className="small-video-id-a">{id}</a>
          </div>
        </div>
      </div>
    );
  };
}

export default SmallVideoView;
