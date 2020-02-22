import React from 'react';
import PropTypes from "prop-types";
import ReactDOM from 'react-dom';
import { Modal, Button, Tooltip, Input,Icon } from 'antd';
import DotsVerticalIcon from "mdi-react/DotsVerticalIcon";

export default class ToolShare extends React.Component {
    constructor(props) {
        super(props);
        this.state =
        {
            visible: false,
            application: 'SFU',
         };
    }
    showModal = () => {
        this.setState({
            visible: true,
        });

        let loginInfo = this.props.loginInfo;
        let host = window.location.host;
        let url = window.location.protocol + "//" + host + "/?room=" + loginInfo.roomId;
        this.setState({ url });
    }
    handleOk = (e) => {
        this.setState({
            visible: false,
        });
    }
    handleCancel = (e) => {
        this.setState({
            visible: false,
        });
    }

    onFocus = (e) => {
        ReactDOM.findDOMNode(e.target).select();
    }

    render() {
        return (
            <div className="app-header-tool-container">
                <Tooltip title='Shared conference'>
                <Button ghost size="large" type="link" onClick={this.showModal}>
                  <Icon
                    component={DotsVerticalIcon}
                    style={{ display: "flex", justifyContent: "center" }}
                  />
                </Button>
                </Tooltip>
                <Modal
                    title='Shared conference'
                    visible={this.state.visible}
                    onOk={this.handleOk}
                    onCancel={this.handleCancel}
                    okText='Ok'
                    cancelText='Cancel'>
                    <div>
                        <div>
                            <span>Send link to your friends</span>
                            <Input onFocus={this.onFocus} readOnly={true} value={this.state.url} />
                        </div>
                    </div>
                </Modal>
            </div>
        );
    }
}

ToolShare.propTypes = {
    roomInfo: PropTypes.any,
}

