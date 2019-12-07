import React from "react";
import { Form, Icon, Input, Button } from "antd";

class LoginForm extends React.Component {

  componentDidMount = () => {
    const { form, loginInfo } = this.props;
    form.setFieldsValue({
      'roomId': loginInfo.roomId,
      'displayName': loginInfo.displayName,
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
