import React from "react";
import { setTimeout } from "timers";
import MicrophoneIcon from "mdi-react/MicrophoneIcon";
import MicrophoneOffIcon from "mdi-react/MicrophoneOffIcon";
import VideoIcon from "mdi-react/VideoIcon";
import VideocamOffIcon from "mdi-react/VideocamOffIcon";

class LocalVideoView extends React.Component {
  componentDidMount = () => {
    const { stream } = this.props;
    this.video.srcObject = stream.stream;
  };

  render = () => {
    const { id, label, audioMuted, videoMuted } = this.props;
    let allHide = !audioMuted && !videoMuted;
    return (
      <div className="local-video-layout">
        <div className="local-video-container">
          <video
            ref={ref => {
              this.video = ref;
            }}
            id={id}
            autoPlay
            playsInline
            muted={true}
            className="local-video-size"
          />
          <div className="local-video-icon-layout">
            {
              allHide ? "" : (audioMuted ? <MicrophoneOffIcon size={18} color="white"/> : <MicrophoneIcon size={18} color="white"/> )
            }
            {
              allHide ? "" : (videoMuted ? <VideocamOffIcon size={18} color="white"/> : <VideoIcon size={18} color="white"/>)
            }
            
          </div>
          <a className="local-video-name">
            {label}
          </a>
        </div>
      </div>
    );
  };
}

export default LocalVideoView;
