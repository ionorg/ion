import React from "react";
import { LocalVideoView, MainVideoView, SmallVideoView } from "./videoview";
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
    this._unpublish();
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
    this.localVideoView.stream = stream;
  };

  _unpublish = async () => {
    const { client } = this.props;
    if (this.localVideoView) this.localVideoView.stream = null;
    await client.unpublish();
  };

  _handleStreamEnabled = (type, enabled) => {
    if (enabled) {
      this._publish(type);
    } else {
      this._unpublish();
    }
  };

  _handleAddStream = async (id, rid) => {
    const { client } = this.props;
    let streams = this.state.streams;
    let stream = await client.subscribe(id);
    streams.push({ id, stream });
    this.setState({ streams });
  };

  _handleRemoveStream = async (id, rid) => {
    let streams = this.state.streams;
    streams = streams.filter(item => item.id !== id);
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
      if (item.id == id) {
        big = i;
        break;
      }
    }

    let c = streams[first];
    streams[first] = streams[big];
    streams[big] = c;

    this.setState({ streams: streams });
    setTimeout(this.replay, 1000);
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
                <MainVideoView
                  key={item.id}
                  id={item.id}
                  stream={item.stream}
                />
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
              ref={ref => (this.localVideoView = ref)}
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
                    <SmallVideoView
                      key={item.id}
                      id={item.id}
                      stream={item.stream}
                      index={index}
                      onClick={this._onChangeVideoPosition}
                    />
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
