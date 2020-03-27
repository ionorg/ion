import React from "react";
import MicrophoneOffIcon from "mdi-react/MicrophoneOffIcon";
import VideocamOffIcon from "mdi-react/VideocamOffIcon";
import { Avatar,Icon,Button } from 'antd';
import PictureInPictureBottomRightOutlineIcon from "mdi-react/PictureInPictureBottomRightOutlineIcon";

class LocalVideoView extends React.Component {

  constructor() {
    super();
    this.state = {
      minimize: false,
    }
  }


  componentDidMount = () => {
    const { stream } = this.props;
    this.video.srcObject = stream.stream;
  };

  componentWillUnmount = () => {
    this.video.srcObject = null;
  }

  onMinimizeClick = () => {
    let minimize = !this.state.minimize;
    this.setState({ minimize });
  }

  render = () => {
    const { id, label, audioMuted, videoMuted,videoType } = this.props;

    let minIconStyle = 'local-video-icon-layout';
    if(videoType == 'localVideo'){
      minIconStyle = 'local-video-min-layout';
    }

    return (
        <div className="local-video-container" style={{ borderWidth: `${this.state.minimize ? '0px' : '0.5px'}` }}>
          <video
            ref={ref => {
              this.video = ref;
            }}
            id={id}
            autoPlay
            playsInline
            muted={true}
            className="local-video-size"
            style={{ display: `${this.state.minimize ? 'none' : ''}` }}
          />
          <div className = {`${this.state.minimize ? minIconStyle : 'local-video-icon-layout'}`}>
            { !this.state.minimize && audioMuted && <MicrophoneOffIcon size={18} color="white" />}
            { !this.state.minimize && videoMuted && <VideocamOffIcon size={18} color="white" />}

            <Button
                  ghost
                  size="small"
                  type="link"
                  onClick={() => this.onMinimizeClick()}
            > 
              <PictureInPictureBottomRightOutlineIcon
                size={18}
              />
          </Button>

          </div>
          {
            videoMuted ?
            <div className="local-video-avatar" style={{ display: `${this.state.minimize ? 'none' : ''}` }}>
              <Avatar size={64} icon="user"/>
            </div>
            : ""
          }
          <a className="local-video-name" style={{ display: `${this.state.minimize ? 'none' : ''}` }}>{label}</a>
        </div>
    );
  };
}

export default LocalVideoView;
