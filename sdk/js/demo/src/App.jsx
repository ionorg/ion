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
const { Header, Content, Footer } = Layout;
import { reactLocalStorage } from "reactjs-localstorage";

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
      })
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
    await this.client.join(values.roomId);
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

  render() {
    const { login, loading, loginInfo } = this.state;
    return (
      <Layout style={{ height: "100%" }}>
        <Header
          style={{
            padding: 0
          }}
        >
          <div style={{ float: "left", marginLeft: 8 }}>
            <a href="https://pion.ly" target="_blank">
              <img
                src="https://pion.ly/img/pion-logo.svg"
                style={{ width: 112, height: 28 }}
              />
            </a>
          </div>
          <div style={{ float: "right", marginRight: 24, color: "#001529" }}>
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

        <Content
          style={{
            width: "100%",
            minHeight: "540px",
            display: "flex",
            justifyContent: "center",
            alignItems: "center"
          }}
        >
          {login ? (
            <Conference client={this.client} />
          ) : loading ? (
            <Spin size="large" tip="Connecting..." />
          ) : (
            <Card title="Join to Ion" style={{ height: 280 }}>
              <LoginForm loginInfo={loginInfo} handleLogin={this._handleJoin} />
            </Card>
          )}
        </Content>

        {!login && (
          <Footer style={{ textAlign: "center" }}>
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
