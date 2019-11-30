import React from "react";
import { Button, Icon, Input } from "antd";

class LocalVideoView extends React.Component {
  state = {
    type: "none",
    playing: false
  };

  _publish = async type => {
    const { client, id } = this.props;
    let stream = await client.publish({
      codec: "vp8",
      audio: true,
      video: type === "video",
      screen: type === "screen"
    });
    let video = this.refs[id];
    video.srcObject = stream.stream;
    this.setState({ playing: true });
  };

  _unpublish = async () => {
    const { client, id } = this.props;
    let video = this.refs[id];
    let stream = video.srcObject;
    if (stream !== null) {
      let tracks = stream.getTracks();
      for (let i = 0, len = tracks.length; i < len; i++) {
        tracks[i].stop();
      }
    }
    video.srcObject = null;
    await client.unpublish();
    this.setState({ playing: false });
  };

  _handleMediaSwitch = (type, playing) => {
    this.setState({ type, playing });
    if (playing) {
      this._publish(type);
    } else {
      this._unpublish();
    }
  };

  render = () => {
    const { id } = this.props;
    const { playing, type } = this.state;
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
        <div style={{ position: "relative", left: 0, right: 0, zIndex: 99 }}>
          <div>
            <a
              style={{
                position: "absolute",
                left: 8,
                top: 8,
                color: "#ffffff"
              }}
            >
              Local
            </a>
            <div style={{ position: "absolute", right: 8, top: 8 }}>
              <div style={{ display: "inline" }}>
                <Button
                  shape="circle"
                  icon="audio"
                  disabled={playing && type != "audio"}
                  onClick={() => this._handleMediaSwitch("audio", !playing)}
                />
                <Button
                  shape="circle"
                  icon="video-camera"
                  disabled={playing && type != "video"}
                  onClick={() => this._handleMediaSwitch("video", !playing)}
                />
                <Button
                  shape="circle"
                  icon="desktop"
                  disabled={playing && type != "screen"}
                  onClick={() => this._handleMediaSwitch("screen", !playing)}
                />
              </div>
            </div>
          </div>
        </div>

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
            muted={true}
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

export default LocalVideoView;
