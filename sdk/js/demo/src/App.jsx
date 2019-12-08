import React from "react";
import {
  Layout,
  Button,
  Modal,
  Icon,
  Input,
  notification,
  Card,
  Spin
} from "antd";

const { confirm } = Modal;
const { Header, Content, Footer,Sider } = Layout;
import { reactLocalStorage } from "reactjs-localstorage";
import MicrophoneIcon from 'mdi-react/MicrophoneIcon';
import MicrophoneOffIcon from 'mdi-react/MicrophoneOffIcon';
import PhoneHangupOutlineIcon from 'mdi-react/PhoneHangupOutlineIcon';
import DesktopMacIcon from 'mdi-react/DesktopMacIcon';
import VideoIcon from 'mdi-react/VideoIcon';
import VideocamOffIcon from 'mdi-react/VideocamOffIcon';

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
    };

    let client = new Client();

    window.onunload = () => {
      client.leave();
    };

    client.on("peer-join", (id, rid) => {
      this._notification("Peer Join", "peer => " + id + ", join!");
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

    client.on("stream-add", (id, rid) => {
      console.log("stream-add %s,%s!", id, rid);
      this._notification("Stream Add", "id => " + id + ", rid => " + rid);
    });

    client.on("stream-remove", (id, rid) => {
      console.log("stream-remove %s,%s!", id, rid);
      this._notification("Stream Remove", "id => " + id + ", rid => " + rid);
    });

    client.init();
    this.client = client;
  }

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
        await client.leave();
        this2.setState({ login: false });
      },
      onCancel() {
        console.log("Cancel");
      }
    });
  };

  _handleStreamEnabled = (type, enabled) => {
    if(type == 'audio'){
      this.setState({
        localAudio:enabled,
      });
    }
    if(type == 'video'){
      this.setState({
        localVideo:enabled,
      });
    }
    if(type == 'screen'){
      this.setState({
        localScreen:enabled,
      });
    }
    this.conference._handleStreamEnabled(type, enabled);
  };

  _onRef = (ref) => {
    this.conference = ref;
  }

  render() {
    const { login, loading, loginInfo,localAudio, localVideo,localScreen } = this.state;
    return (
      <Layout className="app-layout">
        <Header className="app-header">
          <div className="app-header-left">
            <a href="https://pion.ly" target="_blank">
              <img
                src="https://pion.ly/img/pion-logo.svg"
                className="app-logo-img"
              />
            </a>
          </div>
          {
            login?
            (
              <div className="app-header-tool">
              {
                localAudio?
                <MicrophoneIcon className="app-header-tool-button" size={32} 
                onClick={() => this._handleStreamEnabled("audio", !localAudio)}/>
                :
                <MicrophoneOffIcon className="app-header-tool-button" size={32}
                onClick={() => this._handleStreamEnabled("audio", !localAudio)}/>
              }
              <DesktopMacIcon className="app-header-tool-button" size={26}
              onClick={() => this._handleStreamEnabled("screen", !localScreen)}/>
              {
                localVideo ? 
                <VideoIcon className="app-header-tool-button" size={32}
                onClick={() => this._handleStreamEnabled("video", !localVideo)}/>
                :
                <VideocamOffIcon className="app-header-tool-button" size={32}
                onClick={() => this._handleStreamEnabled("video", !localVideo)}/>
              }
              </div>
            ): (<div></div>)
          }
          <div className="app-header-right">
            {login ? (
              <Button
                shape="circle"
                icon="logout"
                ghost
                onClick={this._handleLeave}
              />
            ) : (
              <Button shape="circle" icon="setting" ghost />
            )}
          </div>
        </Header>

        <Content className="app-center-layout">
          {login ? (
            
            <Layout className="app-content-layout">
              <Sider 
                width={320}
                style={{ background: '#f5f5f5' }}
                collapsedWidth={0}
                trigger={null}
                collapsible
                collapsed={false}>
              </Sider>
              <Layout className="app-right-layout">
                <Content style={{ flex: 1 }}>
                  <Conference onRef={this._onRef} client={this.client} />
                </Content>
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
