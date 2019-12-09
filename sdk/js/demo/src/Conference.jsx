import React from "react";
import {
  LocalVideoView,
  MainVideoView,
  SmallVideoView,
  LocalScreenView
} from "./videoview";

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
    return stream;
  };

  unpublish = async () => {
    const { client } = this.props;
    if (this.localVideoView) this.localVideoView.stream = null;
    if (this.localVideoStream) {
      await client.unpublish(this.localVideoStream.mid);
      this.localVideoStream = null;
    }
    if (this.localScreenView) {
      await client.unpublish(this.localScreenView.mid);
      this.localScreenView = null;
    }
  };

  handleStreamEnabled = async (type, enabled) => {
    if (type === "screen") {
      if (enabled) {
        let stream = await this._publish(type);
        this.localScreenStream = stream;
        this.localScreenView.stream = stream;
      } else {
        if (this.localScreenStream) {
          const { client } = this.props;
          await client.unpublish(this.localScreenStream.mid);
          this.localScreenStream = null;
          this.localScreenView.stream = null;
        }
      }
    } else {
      if (enabled) {
        let stream = await this._publish(type);
        this.localVideoStream = stream;
        this.localVideoView.stream = stream;
      } else {
        if (this.localVideoStream) {
          const { client } = this.props;
          await client.unpublish(this.localVideoStream.mid);
          this.localVideoStream = null;
          this.localVideoView.stream = null;
        }
      }
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
    setTimeout(this.replay, 1000);
  };

  render = () => {
    const { client } = this.props;
    const { streams } = this.state;
    var id = client.uid;
    const { clientWidth, clientHeight } = this.state;
    return (
      <div className="conference-layout" ref={this.saveRef}>
        {streams.map((item, index) => {
          return index == 0 ? (
            <MainVideoView key={item.mid} id={item.mid} stream={item.stream} />
          ) : (
            ""
          );
        })}
        <div className="conference-local-video-layout">
          <div className="conference-local-video-size">
            <LocalVideoView
              id={id + "-video"}
              ref={ref => {
                this.localVideoView = ref;
              }}
              client={client}
            />
          </div>
        </div>
        <div className="conference-local-screen-layout">
          <div className="conference-local-video-size">
            <LocalScreenView
              id={id + "-screen"}
              ref={ref => {
                this.localScreenView = ref;
              }}
              client={client}
            />
          </div>
        </div>
        {streams.map((item, index) => {
          return index > 0 ? (
            <SmallVideoView
              key={item.mid}
              id={item.mid}
              stream={item.stream}
              index={index}
              onClick={this._onChangeVideoPosition}
            />
          ) : (
            ""
          );
        })}
      </div>
    );
  };
}

export default Conference;
