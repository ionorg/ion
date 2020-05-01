import React, { Component } from 'react';
import { Icon } from 'antd';
import UserIcon from "mdi-react/UserIcon";

export default class ChatBubble extends Component {
  constructor(props) {
    super()
    
    this.state = {message: props.message};
  }

  componentDidMount() {}

  render() {


    if (this.props.message.id == 1){
      return (
          <div className='bubble-left'>
            <div className='bubble-head'>
              <Icon component={UserIcon} />
            </div>
            <div className='bubble-msg'>
              <p className="sender-name">{this.props.message.senderName}</p>
              <div className='bubble-msgword'>
                <p className='pl'>
                  {this.props.message.message}
                </p>
              </div>
            </div>
          </div>
      )
    }
    else if(this.props.message.id == 0){
      return (
          <div className='bubble-right'>

            <div className='bubble-msg'>
                <p style={{textAlign:'right'}} className="sender-name">{this.props.message.senderName}</p>
                <div className='bubble-msgword'>
                  <p className='pr'>
                    {this.props.message.message}
                  </p>
                </div>
            </div>

            <div className='bubble-head'>
              <Icon component={UserIcon} />
            </div>

          </div>
      )
    }
    else if(this.props.message.id == 2){
        return (
            <div className='bubble-middle'>

                <div className='bubble-msg'>
                    <div className='bubble-msgword-middle'>
                        <p className='pm'>
                            {this.props.message.message}
                        </p>
                    </div>
                </div>
            </div>
        )
    }
  }
}
