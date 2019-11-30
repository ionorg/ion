import React from "react";
import {
  Layout,
  Button,
  Menu,
  Icon,
  Input,
  notification,
  Card,
  Spin
} from "antd";

const { Header, Content, Footer } = Layout;
const { SubMenu } = Menu;

import LoginForm from "./LoginForm";
import Conference from "./Conference";
import { Client, Stream } from "ion-sdk";

class App extends React.Component {
  constructor() {
    super();
    this.state = {
      login: false,
      loading: false
    };

    let client = new Client();

    window.onunload = () => {
      client.leave();
    };
    
    client.on("peer-join", (id, rid) => {
      notification("Peer Join", "peer => " + id + ", join!");
    });

    client.on("peer-leave", (id, rid) => {
      notification("Peer Leave", "peer => " + id + ", leave!");
    });

    client.on("transport-open", function() {
      console.log("transport open!");
    });

    client.on("transport-closed", function() {
      console.log("transport closed!");
    });

    client.init();
    this.client = client;
  }

  notification = (message, description) => {
    notification.open({
      message: message,
      description: description,
      icon: <Icon type="smile" style={{ color: "#108ee9" }} />
    });
  };

  handleJoin = async values => {
    this.setState({ loading: true });
    await this.client.join(values.room_id);
    this.setState({ login: true, loading: false });
    this.notification(
      "Connected!",
      "Welcome to the ion room => " + values.room_id
    );
  };

  render() {
    const { login, loading } = this.state;
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
              <Button shape="circle" icon="logout" ghost />
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
              <LoginForm handleLogin={this.handleJoin} />
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
