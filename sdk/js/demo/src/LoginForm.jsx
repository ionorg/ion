import React from "react";
import { Form, Icon, Input, Button } from "antd";

class LoginForm extends React.Component {
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
      <Form onSubmit={this.handleSubmit} style={{ minWidth: 300 }}>
        <Form.Item>
          {getFieldDecorator("room_id", {
            rules: [{ required: true, message: "Please enter your room Id!" }]
          })(
            <Input
              prefix={<Icon type="team" style={{ color: "rgba(0,0,0,.25)" }} />}
              placeholder="Room Id"
            />
          )}
        </Form.Item>
        <Form.Item>
          {getFieldDecorator("display_name", {
            rules: [{ required: true, message: "Please enter your Name!" }]
          })(
            <Input
              prefix={
                <Icon type="contacts" style={{ color: "rgba(0,0,0,.25)" }} />
              }
              type="display_name"
              placeholder="Display Name"
            />
          )}
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" style={{ width: "100%" }}>
            Join
          </Button>
        </Form.Item>
      </Form>
    );
  }
}

const WrappedLoginForm = Form.create({ name: "normal_login" })(LoginForm);
export default WrappedLoginForm;
