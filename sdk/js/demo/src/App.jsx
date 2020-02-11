import React from "react";
import { Layout, Button, Modal, Icon, notification, Card, Spin,Tooltip } from "antd";
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
import MediaSettings from './settings';

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
      localAudioEnabled: true,
      localVideoEnabled: true,
      screenSharingEnabled: false,
      collapsed: true,
      isFullScreen:false
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
    notification.info({
      message: message,
      description: description,
      placement: 'bottomRight',
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
    await this.conference.handleLocalStream(true);
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

  _handleAudioTrackEnabled = enabled => {
    this.setState({
      localAudioEnabled: enabled
    });
    this.conference.muteMediaTrack("audio", enabled);
  };

  _handleVideoTrackEnabled = enabled => {
    this.setState({
      localVideoEnabled: enabled
    });
    this.conference.muteMediaTrack("video", enabled);
  };

  _handleScreenSharing = enabled => {
    this.setState({
      screenSharingEnabled: enabled
    });
    this.conference.handleScreenSharing(enabled);
  };

  _onRef = ref => {
    this.conference = ref;
  };

  _openOrCloseLeftContainer = collapsed => {
    this.setState({
      collapsed: collapsed
    });
  };

  _onFullScreenClickHandler = () => {
    let docElm = document.documentElement;

    if (this._fullscreenState()) {

        if (document.exitFullscreen) {
            document.exitFullscreen();
        }
        else if (document.mozCancelFullScreen) {
            document.mozCancelFullScreen();
        }
        else if (document.webkitCancelFullScreen) {
            document.webkitCancelFullScreen();
        }
        else if (document.msExitFullscreen) {
            document.msExitFullscreen();
        }

        this.setState({ isFullScreen: false });

    } else {
        if (docElm.requestFullscreen) {
            docElm.requestFullscreen();
        }
        //FireFox
        else if (docElm.mozRequestFullScreen) {
            docElm.mozRequestFullScreen();
        }
        //Chrome等
        else if (docElm.webkitRequestFullScreen) {
            docElm.webkitRequestFullScreen();
        }
        //IE11
        else if (elem.msRequestFullscreen) {
            elem.msRequestFullscreen();
        }

        this.setState({ isFullScreen: true });
    }
  }

  _fullscreenState = () => {
      return document.fullscreen ||
          document.webkitIsFullScreen ||
          document.mozFullScreen ||
          false;
  }

  _onDeviceSelectedChanged = (selectedAudioDevice,selectedVideoDevice,resolution,bandwidth) => {
    //TODO
  }

  render() {
    const {
      login,
      loading,
      loginInfo,
      localAudioEnabled,
      localVideoEnabled,
      screenSharingEnabled,
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
              <Tooltip title='Mute/Cancel'>
              <Button
                ghost
                size="large"
                style={{ color: localAudioEnabled ? "" : "red" }}
                type="link"
                onClick={() =>
                  this._handleAudioTrackEnabled(!localAudioEnabled)
                }
              >
                <Icon
                  component={
                    localAudioEnabled ? MicrophoneIcon : MicrophoneOffIcon
                  }
                  style={{ display: "flex", justifyContent: "center" }}
                />
              </Button>
              </Tooltip>
              <Tooltip title='Open/Close video'>
                <Button
                  ghost
                  size="large"
                  style={{ color: localVideoEnabled ? "" : "red" }}
                  type="link"
                  onClick={() =>
                    this._handleVideoTrackEnabled(!localVideoEnabled)
                  }
                >
                  <Icon
                    component={localVideoEnabled ? VideoIcon : VideocamOffIcon}
                    style={{ display: "flex", justifyContent: "center" }}
                  />
                </Button>
              </Tooltip>
              <Tooltip title='Hangup'>
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
              </Tooltip>
              <Tooltip title='Share desktop'>
                <Button
                  ghost
                  size="large"
                  type="link"
                  style={{ color: screenSharingEnabled ? "red" : "" }}
                  onClick={() => this._handleScreenSharing(!screenSharingEnabled)}
                >
                  <Icon
                    component={
                      screenSharingEnabled ? TelevisionOffIcon : TelevisionIcon
                    }
                    style={{ display: "flex", justifyContent: "center" }}
                  />
                </Button>
              </Tooltip>
              <Tooltip title='Shared conference'>
                <Button ghost size="large" type="link">
                  <Icon
                    component={DotsVerticalIcon}
                    style={{ display: "flex", justifyContent: "center" }}
                  />
                </Button>
              </Tooltip>
            </div>
          ) : (
            <div />
          )}
          <div className="app-header-right">
            <MediaSettings onInputDeviceSelected={this._onDeviceSelectedChanged} />
          </div>
        </Header>

        <Content className="app-center-layout">
          {login ? (
            <Layout className="app-content-layout">
              <Sider
                width={320}
                style={{ background: "#333" }}
                collapsedWidth={0}
                trigger={null}
                collapsible
                collapsed={this.state.collapsed}
              />
              <Layout className="app-right-layout">
                <Content style={{ flex: 1 }}>
                  <Conference
                    collapsed={this.state.collapsed}
                    client={this.client}
                    ref={ref => {
                      this.conference = ref;
                    }}
                  />
                </Content>
                <div className="app-collapsed-button">
                  <Tooltip title='Open/Close chat panel'>
                    <Button
                      icon={this.state.collapsed ? "right" : "left"}
                      size="large"
                      shape="circle"
                      ghost
                      onClick={() => this._openOrCloseLeftContainer(!collapsed)}
                    />
                  </Tooltip>
                </div>  
                <div className="app-fullscreen-button">
                  <Tooltip title='Fullscreen/Exit'>
                    <Button
                      icon={this.state.isFullScreen ? "fullscreen-exit" : "fullscreen"}
                      size="large"
                      shape="circle"
                      ghost
                      onClick={() => this._onFullScreenClickHandler()}
                    />
                  </Tooltip>
                </div>
                
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
