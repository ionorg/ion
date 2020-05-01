'use strict'

import React, { Component } from 'react'
import ReactDOM from 'react-dom'
import PropTypes from 'prop-types';
import ChatBubble from './chatbubble';
import ChatInput from './chatinput';

export default class ChatFeed extends Component {
  constructor(props) {
    super(props)
    this.state = {
      messages: props.messages || [],
    }
  }

  _scrollToBottom() {
    const { chat } = this.refs;
    if(chat !== undefined){
      const scrollHeight = chat.scrollHeight;
      const height = chat.clientHeight;
      const maxScrollTop = scrollHeight - height;
      ReactDOM.findDOMNode(chat).scrollTop = maxScrollTop > 0 ? maxScrollTop : 0;
    }
  }

  _renderGroup(messages, index, id) {
    var group = []

    for (var i = index; messages[i] ? messages[i].id == id : false; i--) {
      group.push(messages[i])
    }

    var message_nodes = group.reverse().map((curr, index) => {
      return (
        <ChatBubble
          key={Math.random().toString(36)}
          message={curr}
        />
      )
    })
    return (
      <div key={Math.random().toString(36)} className='chatbubble-wrapper'>
        {message_nodes}
      </div>
    )
  }

  _renderMessages(messages) {
    var message_nodes = messages.map((curr, index) => {
      // Find diff in message type or no more messages
      if (
        (messages[index + 1] ? false : true) ||
        messages[index + 1].id != curr.id
      ) {
        return this._renderGroup(messages, index, curr.id)
      }
    })
    // return nodes
    return message_nodes
  }

  render() {
    window.setTimeout(() => {
      this._scrollToBottom()
    }, 10)

    const messages = [
      {id:0,message:"hello every one",senderName:'kevin kang'},
      ];

    return (
      <div id="chat-panel" className='chat-panel'>

        <div className='title-panel'>
          <span className='title-chat'>Chat</span>
        </div>

        <div ref="chat" className='chat-history'>
          <div>
            {this._renderMessages(this.props.messages)}
          </div>
        </div>
        <ChatInput onSendMessage={this.props.onSendMessage}/>
      </div>
    )
  }
}


ChatFeed.propTypes = {
  isTyping: PropTypes.bool,
  messages: PropTypes.array.isRequired,
  onSendMessage: PropTypes.func.isRequired,
}
