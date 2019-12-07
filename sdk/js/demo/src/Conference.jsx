import React from "react";
import MainVideoView from "./MainVideoView";
import LocalVideoView from "./LocalVideoView";
import SmallVideoView from "./SmallVideoView";
import styles from "../styles/scss/styles.scss";

class Conference extends React.Component {
  constructor() {
    super();
    this.state = {
      streams: [],
      clientWidth: 0,
      clientHeight: 0
    };
    this.saveRef = ref => {
      if (ref !== null) {
        const { clientWidth, clientHeight } = ref;
        this.refDom = ref;
        this.setState({ clientWidth, clientHeight });
      }
    };
  }

  componentDidMount = () => {
    const { client } = this.props;
    client.on("stream-add", this._handleAddStream);
    client.on("stream-remove", this._handleRemoveStream);
    window.addEventListener("resize", this._onWindowResize);
  };

  componentWillUnmount = () => {
    const { client } = this.props;
    client.off("stream-add", this._handleAddStream);
    client.off("stream-remove", this._handleAddStream);
    window.removeEventListener("resize", this._onWindowResize);
  };

  _onWindowResize = () => {
    const { clientWidth, clientHeight } = this.refDom;
    this.setState({ clientWidth, clientHeight });
  };

  _publish = async type => {
    const { client, id } = this.props;
    let stream = await client.publish({
      codec: "vp8",
      audio: true,
      video: type === "video",
      screen: type === "screen"
    });
    this.localStream = stream;
    this.localVideoView.stream = stream;
  };

  unpublish = async () => {
    const { client } = this.props;
    if (this.localVideoView) this.localVideoView.stream = null;
    if (this.localStream) {
      await client.unpublish(this.localStream.mid);
      this.localStream = null;
    }
  };

  _handleStreamEnabled = (type, enabled) => {
    if (enabled) {
      this._publish(type);
    } else {
      this.unpublish();
    }
  };

  _handleAddStream = async (rid, mid) => {
    const { client } = this.props;
    let streams = this.state.streams;
    let stream = await client.subscribe(rid, mid);
    streams.push({ mid, stream });
    this.setState({ streams });
  };

  _handleRemoveStream = async (rid, mid) => {
    let streams = this.state.streams;
    streams = streams.filter(item => item.mid !== mid);
    this.setState({ streams });
  };

  render = () => {
    const { client } = this.props;
    const { streams } = this.state;
    var id = client.uid;
    const { clientWidth, clientHeight } = this.state;
    return (
      <div style={{ width: "100%", height: "100%" }} ref={this.saveRef}>
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            justifyContent: "center",
            alignContent: "center"
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "center",
              width: clientWidth,
              height: clientHeight - 64
            }}
          >
            {streams.map((item, index) => {
              return index == 0 ? (
                <MainVideoView id={item.id} stream={item.stream} />
              ) : (
                ""
              );
            })}
          </div>
        </div>
        <div style={{ position: "absolute", top: 90, left: 24 }}>
          <div style={{ width: 220, height: 140 }}>
            <LocalVideoView
              id={id}
              ref={ref => {
                this.localVideoView = ref;
              }}
              client={client}
              handleStreamEnabled={this._handleStreamEnabled}
            />
          </div>
        </div>
        <div style={{ position: "absolute", left: 16, right: 16, bottom: 24 }}>
          <div
            style={{
              position: "relative",
              display: "block",
              width: clientWidth
            }}
          >
            <div style={{ whiteSpace: "nowrap", overflow: "scroll" }}>
              {streams.map((item, index) => {
                return index > 0 ? (
                  <div
                    style={{ display: "inline-block", width: 220, height: 140 }}
                  >
                    <SmallVideoView id={item.id} stream={item.stream} />
                  </div>
                ) : (
                  ""
                );
              })}
            </div>
          </div>
        </div>
      </div>
    );
  };
}

export default Conference;
