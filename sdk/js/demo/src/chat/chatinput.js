import React, { Component } from "react";
import PropTypes from 'prop-types';
import { Input,Button } from 'antd';

export default class ChatInput extends Component {
  constructor(props) {
    super(props);

    this.state = {
      inputMessage:"",
    }
  }

  onInputChange = (event) => {
    this.setState({
      inputMessage:event.target.value,
    });
  }

  onBtnSendHandler = (event) => {
    this.sendMessage();
  }


  onInputKeyUp = (event) => {
    if (event.keyCode == 13) {
      this.sendMessage();
    }
  }

  sendMessage = () =>{
    let msg = this.state.inputMessage;

    if (msg.length === 0) {
      return;
    }
    if (msg.replace(/(^\s*)|(\s*$)/g, "").length === 0) {
      return;
    }
    this.props.onSendMessage(msg);
    this.setState({
      inputMessage:"",
    });
  }

  render() {
    return (
        <div className='chat-input'>

          <Input
              placeholder='Please input message'
              onChange={this.onInputChange}
              onPressEnter={this.onInputKeyUp}
              value={this.state.inputMessage}/>
          <Button style={{marginLeft:'4px',}} icon='message' onClick={this.onBtnSendHandler}/>
        </div>
    );
  }
}

ChatInput.propTypes = {
  onSendMessage: PropTypes.func.isRequired,
}
