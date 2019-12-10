import React from "react";
import { Layout, Button, Modal, Icon, notification, Card, Spin } from "antd";
const { confirm } = Modal;
const { Header, Content, Footer, Sider } = Layout;
import { reactLocalStorage } from "reactjs-localstorage";
import MicrophoneIcon from "mdi-react/MicrophoneIcon";
import MicrophoneOffIcon from "mdi-react/MicrophoneOffIcon";
import HangupIcon from "mdi-react/PhoneHangupIcon";
import TelevisionIcon from "mdi-react/TelevisionIcon";
import TelevisionOffIcon from "mdi-react/TelevisionOffIcon";
import VideoIcon from "mdi-react/VideoIcon";
import VideocamOffIcon from "mdi-react/VideocamOffIcon";
import DotsVerticalIcon from "mdi-react/DotsVerticalIcon";

import LoginForm from "./LoginForm";
import Conference from "./Conference";
import { Client, Stream } from "ion-sdk";

class App extends React.Component {
  constructor() {
    super();
    this.state = {
      login: false,
      loading: false,
      loginInfo: reactLocalStorage.getObject("loginInfo", {
        roomId: "room1",
        displayName: "Guest"
      }),
      localAudio: true,
      localVideo: true,
      localScreen: false,
      collapsed: true
    };

    let client = new Client();

    window.onunload = async () => {
      await this._cleanUp();
    };

    client.on("peer-join", (id, rid, info) => {
      this._notification("Peer Join", "peer => " + info.name + ", join!");
    });

    client.on("peer-leave", (id, rid) => {
      this._notification("Peer Leave", "peer => " + id + ", leave!");
    });

    client.on("transport-open", function() {
      console.log("transport open!");
    });

    client.on("transport-closed", function() {
      console.log("transport closed!");
    });

    client.on("stream-add", (id, rid, info) => {
      console.log("stream-add %s,%s!", id, rid);
      this._notification(
        "Stream Add",
        "id => " + id + ", rid => " + rid + ", name => " + info.name
      );
    });

    client.on("stream-remove", (id, rid) => {
      console.log("stream-remove %s,%s!", id, rid);
      this._notification("Stream Remove", "id => " + id + ", rid => " + rid);
    });

    client.init();
    this.client = client;
  }

  _cleanUp = async () => {
    await this.conference.cleanUp();
    await this.client.leave();
  };

  _notification = (message, description) => {
    notification.open({
      message: message,
      description: description,
      icon: <Icon type="smile" style={{ color: "#108ee9" }} />
    });
  };

  _handleJoin = async values => {
    this.setState({ loading: true });
    reactLocalStorage.clear("loginInfo");
    reactLocalStorage.setObject("loginInfo", values);
    await this.client.join(values.roomId, { name: values.displayName });
    this.setState({ login: true, loading: false });
    this._notification(
      "Connected!",
      "Welcome to the ion room => " + values.roomId
    );
  };

  _handleLeave = async () => {
    let client = this.client;
    let this2 = this;
    confirm({
      title: "Leave Now?",
      content: "Do you want to leave the room?",
      async onOk() {
        console.log("OK");
        await this2._cleanUp();
        this2.setState({ login: false });
      },
      onCancel() {
        console.log("Cancel");
      }
    });
  };

  _handleMediaStreamSwitch = (type, enabled) => {
    if (type == "audio") {
      this.setState({
        localAudio: enabled
      });
    }
    if (type == "video") {
      this.setState({
        localVideo: enabled
      });
    }
    if (type == "screen") {
      this.setState({
        localScreen: enabled
      });
    }
    this.conference.handleMediaStreamSwitch(type, enabled);
  };

  _onRef = ref => {
    this.conference = ref;
  };

  _openOrCloseLeftContainer = collapsed => {
    this.setState({
      collapsed: collapsed
    });
  };

  render() {
    const {
      login,
      loading,
      loginInfo,
      localAudio,
      localVideo,
      localScreen,
      collapsed
    } = this.state;
    return (
      <Layout className="app-layout">
        <Header className="app-header">
          <div className="app-header-left">
            <a href="https://pion.ly" target="_blank">
              <img src="/pion-logo.svg" className="app-logo-img" />
            </a>
          </div>
          {login ? (
            <div className="app-header-tool">
              <Button
                ghost
                size="large"
                style={{ color: localAudio ? "" : "red" }}
                type="link"
                onClick={() =>
                  this._handleMediaStreamSwitch("audio", !localAudio)
                }
              >
                <Icon
                  component={localAudio ? MicrophoneIcon : MicrophoneOffIcon}
                  style={{ display: "flex", justifyContent: "center" }}
                />
              </Button>

              <Button
                ghost
                size="large"
                style={{ color: localVideo ? "" : "red" }}
                type="link"
                onClick={() =>
                  this._handleMediaStreamSwitch("video", !localVideo)
                }
              >
                <Icon
                  component={localVideo ? VideoIcon : VideocamOffIcon}
                  style={{ display: "flex", justifyContent: "center" }}
                />
              </Button>

              <Button
                shape="circle"
                ghost
                size="large"
                type="danger"
                style={{ marginLeft: 16, marginRight: 16 }}
                onClick={this._handleLeave}
              >
                <Icon
                  component={HangupIcon}
                  style={{ display: "flex", justifyContent: "center" }}
                />
              </Button>
              <Button
                ghost
                size="large"
                type="link"
                style={{ color: localScreen ? "red" : "" }}
                onClick={() =>
                  this._handleMediaStreamSwitch("screen", !localScreen)
                }
              >
                <Icon
                  component={localScreen ? TelevisionOffIcon : TelevisionIcon}
                  style={{ display: "flex", justifyContent: "center" }}
                />
              </Button>
              <Button ghost size="large" type="link">
                <Icon
                  component={DotsVerticalIcon}
                  style={{ display: "flex", justifyContent: "center" }}
                />
              </Button>
            </div>
          ) : (
            <div />
          )}
          <div className="app-header-right">
            <Button shape="circle" icon="setting" ghost />
          </div>
        </Header>

        <Content className="app-center-layout">
          {login ? (
            <Layout className="app-content-layout">
              <Sider
                width={320}
                style={{ background: "#f5f5f5" }}
                collapsedWidth={0}
                trigger={null}
                collapsible
                collapsed={this.state.collapsed}
              />
              <Layout className="app-right-layout">
                <Content style={{ flex: 1 }}>
                  <Conference
                    client={this.client}
                    ref={ref => {
                      this.conference = ref;
                    }}
                  />
                </Content>
                <Button
                  style={{ margin: 16 }}
                  icon={this.state.collapsed ? "left" : "right"}
                  size="large"
                  shape="circle"
                  ghost
                  onClick={() => this._openOrCloseLeftContainer(!collapsed)}
                />
              </Layout>
            </Layout>
          ) : loading ? (
            <Spin size="large" tip="Connecting..." />
          ) : (
            <Card title="Join to Ion" className="app-login-card">
              <LoginForm loginInfo={loginInfo} handleLogin={this._handleJoin} />
            </Card>
          )}
        </Content>

        {!login && (
          <Footer className=".app-footer">
            Powered by{" "}
            <a href="https://pion.ly" target="_blank">
              Pion
            </a>{" "}
            WebRTC.
          </Footer>
        )}
      </Layout>
    );
  }
}

export default App;
