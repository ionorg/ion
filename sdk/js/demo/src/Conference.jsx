import React from "react";
import { Spin } from "antd";
import { LocalVideoView, MainVideoView, SmallVideoView } from "./videoview";

class Conference extends React.Component {
  constructor() {
    super();
    this.state = {
      streams: [],
      localStream: null,
      localScreen: null,
      audioMuted: false,
      videoMuted: false
    };
  }

  componentDidMount = () => {
    const { client } = this.props;
    client.on("stream-add", this._handleAddStream);
    client.on("stream-remove", this._handleRemoveStream);
  };

  componentWillUnmount = () => {
    const { client } = this.props;
    client.off("stream-add", this._handleAddStream);
    client.off("stream-remove", this._handleRemoveStream);
  };

  _publish = async (type, codec) => {
    const { client, settings } = this.props;
    let stream = await client.publish({
      codec: settings.codec,
      resolution: settings.resolution,
      bandwidth: settings.bandwidth,
      audio: true,
      video: type === "video",
      screen: type === "screen"
    });
    return stream;
  };

  cleanUp = async () => {
    let { localStream, localScreen, streams } = this.state;
    await this.setState({ localStream: null, localScreen: null, streams: [] });

    streams.map(async item => {
      await this._unsubscribe(item);
    });

    if (localStream) await this._unpublish(localStream);
    if (localScreen) await this._unpublish(localScreen);
  };

  _notification = (message, description) => {
    notification.info({
      message: message,
      description: description,
      placement: "bottomRight"
    });
  };

  _unpublish = async stream => {
    const { client } = this.props;
    if (stream) {
      await this._stopMediaStream(stream);
      await client.unpublish(stream.mid);
    }
  };

  _unsubscribe = async item => {
    const { client } = this.props;
    if (item) {
      item.stop();
      await client.unsubscribe(item.rid, item.mid);
    }
  };

  muteMediaTrack = (type, enabled) => {
    let { localStream } = this.state;
    let tracks = localStream.stream.getTracks();
    let track = tracks.find(track => track.kind === type);
    if (track) {
      track.enabled = enabled;
    }
    if (type === "audio") {
      this.setState({ audioMuted: !enabled });
    } else if (type === "video") {
      this.setState({ videoMuted: !enabled });
    }
  };

  handleLocalStream = async (enabled) => {
    let { localStream } = this.state;

    try {
      if (enabled) {
        localStream = await this._publish("video");
      } else {
        if (localStream) {
          this._unpublish(localStream);
          localStream = null;
        }
      }
      this.setState({ localStream });
    } catch (e) {
      console.log("handleLocalStream error => " + e);
      _notification("publish/unpublish failed!", e);
    }

    //Check audio only conference
    this.muteMediaTrack("video", this.props.localVideoEnabled);

  };

  handleScreenSharing = async enabled => {
    let { localScreen } = this.state;
    if (enabled) {
      localScreen = await this._publish("screen");
      let track = localScreen.stream.getVideoTracks()[0];
      if (track) {
        track.addEventListener("ended", () => {
          this.handleScreenSharing(false);
        });
      }
    } else {
      if (localScreen) {
        this._unpublish(localScreen);
        localScreen = null;
      }
    }
    this.setState({ localScreen });
  };

  _stopMediaStream = async (stream) => {
    let mstream =  stream.stream;
    let tracks = mstream.getTracks();
    for (let i = 0, len = tracks.length; i < len; i++) {
      await tracks[i].stop();
    }
  };

  _handleAddStream = async (rid, mid, info) => {
    const { client } = this.props;
    let streams = this.state.streams;
    let stream = await client.subscribe(rid, mid);
    stream.info = info;
    streams.push({ mid: stream.mid, stream, rid, sid: mid });
    this.setState({ streams });
  };

  _handleRemoveStream = async (rid, mid) => {
    let streams = this.state.streams;
    streams = streams.filter(item => item.sid !== mid);
    this.setState({ streams });
  };

  _onChangeVideoPosition = data => {
    let id = data.id;
    let index = data.index;
    console.log("_onChangeVideoPosition id:" + id + "  index:" + index);

    if (index == 0) {
      return;
    }

    const streams = this.state.streams;
    let first = 0;
    let big = 0;
    for (let i = 0; i < streams.length; i++) {
      let item = streams[i];
      if (item.mid == id) {
        big = i;
        break;
      }
    }

    let c = streams[first];
    streams[first] = streams[big];
    streams[big] = c;

    this.setState({ streams: streams });
  };

  render = () => {
    const { client } = this.props;
    const {
      streams,
      localStream,
      localScreen,
      audioMuted,
      videoMuted
    } = this.state;
    const id = client.uid;
    return (
      <div className="conference-layout">
        {streams.length === 0 && (
          <div className="conference-layout-wating">
            <Spin size="large" tip="Wait for other people joining ..." />
          </div>
        )}
        {streams.map((item, index) => {
          return index == 0 ? (
            <MainVideoView key={item.mid} id={item.mid} stream={item.stream} />
          ) : (
            ""
          );
        })}
        {localStream && (
          <div className="conference-local-video-layout">
              <LocalVideoView
                id={id + "-video"}
                label="Local Stream"
                client={client}
                stream={localStream}
                audioMuted={audioMuted}
                videoMuted={videoMuted}
                videoType="localVideo"
              />
            </div>
        )}
        {localScreen && (
          <div className="conference-local-screen-layout">
              <LocalVideoView
                id={id + "-screen"}
                label="Screen Sharing"
                client={client}
                stream={localScreen}
                audioMuted={false}
                videoMuted={false}
                videoType="localScreen"
              />
          </div>
        )}
        <div className="small-video-list-div">
          <div className="small-video-list">
            {streams.map((item, index) => {
              return index > 0 ? (
                <SmallVideoView
                  key={item.mid}
                  id={item.mid}
                  stream={item.stream}
                  videoCount={streams.length}
                  collapsed={this.props.collapsed}
                  index={index}
                  onClick={this._onChangeVideoPosition}
                />
              ) : (
                <div />
              );
            })}
          </div>
        </div>
      </div>
    );
  };
}

export default Conference;
