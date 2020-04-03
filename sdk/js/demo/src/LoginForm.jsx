import React from "react";
import { Form, Icon, Input, Button,Checkbox } from "antd";
import { reactLocalStorage } from "reactjs-localstorage";

class LoginForm extends React.Component {

  componentDidMount = () => {
    const {form} = this.props;
    console.log("window.location:" + window.location);
    console.log("url:" + window.location.protocol + window.location.host + "  " +  window.location.pathname + window.location.query);

    let params = this.getRequest();

    let roomId = 'room1';
    let displayName = 'Guest';
    let audioOnly = false;

    let localStorage = reactLocalStorage.getObject("loginInfo");

    if(localStorage){
      roomId = localStorage.roomId;
      displayName = localStorage.displayName;
      audioOnly = localStorage.audioOnly;
      console.log('localStorage:' + roomId + ' ' + displayName);
    }

    if (params && params.hasOwnProperty('room')) {
      roomId = params.room;
    }

    form.setFieldsValue({
      'roomId': roomId,
      'displayName': displayName,
      'audioOnly': audioOnly,
    });
  }

  handleSubmit = e => {
    e.preventDefault();
    this.props.form.validateFields((err, values) => {
      if (!err) {
        const handleLogin = this.props.handleLogin;
        handleLogin(values);
        console.log("Received values of form: ", values);
      }
    });
  };

  getRequest() {
    let url = location.search; 
    let theRequest = new Object();
    if (url.indexOf("?") != -1) {
      let str = url.substr(1);
      let strs = str.split("&");
      for (let i = 0; i < strs.length; i++) {
        theRequest[strs[i].split("=")[0]] = decodeURI(strs[i].split("=")[1]);
      }
    }
    return theRequest;
  }

  render() {
    const { getFieldDecorator } = this.props.form;

    return (
      <Form onSubmit={this.handleSubmit} className="login-form">
        <Form.Item>
          {getFieldDecorator("roomId", {
            rules: [{ required: true, message: "Please enter your room Id!" }]
          })(
            <Input
              prefix={<Icon type="team" className="login-input-icon" />}
              placeholder="Room Id"
            />
          )}
        </Form.Item>
        <Form.Item>
          {getFieldDecorator("displayName", {
            rules: [{ required: true, message: "Please enter your Name!" }]
          })(
            <Input
              prefix={
                <Icon type="contacts" className="login-input-icon" />
              }
              placeholder="Display Name"
            />
          )}
        </Form.Item>
        <Form.Item>
          {getFieldDecorator('audioOnly', {
            valuePropName: 'checked',
            initialValue: true,
          })(
            <Checkbox>
              Audio only
            </Checkbox>
          )}
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" className="login-join-button">
            Join
          </Button>
        </Form.Item>
      </Form>
    );
  }
}

const WrappedLoginForm = Form.create({ name: "login" })(LoginForm);
export default WrappedLoginForm;
