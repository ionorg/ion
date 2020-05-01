import 'package:flutter/material.dart';
import 'package:date_format/date_format.dart';
import 'chat_message.dart';
import '../utils/random_string.dart';
import 'package:flutter_ion/flutter_ion.dart';


class ChatPage extends StatefulWidget {

  Client client;
  var _historyMessage = [];
  String _displayName;
  String _room;

  ChatPage(this.client, this._historyMessage, this._displayName,this._room);

  @override
  State createState() => ChatPageState();
}

class ChatPageState extends State<ChatPage>{

  Client client;
  var _historyMessage = [];

  String _displayName = "";
  String _room = "";

  final TextEditingController textEditingController = TextEditingController();
  List<ChatMessage> _messages = <ChatMessage>[];

  FocusNode _focusNodeFirstName;

  @override
  void initState() {
    super.initState();

    _focusNodeFirstName = FocusNode();

    client = widget.client;
    _historyMessage = widget._historyMessage;
    _displayName = widget._displayName;
    _room = widget._room;

    if (client != null) {
      for (int i = 0; i < _historyMessage.length; i++) {
        var hisMsg = _historyMessage[i];

        ChatMessage message = ChatMessage(
            randomNumeric(6),
            hisMsg['text'],
            hisMsg['name'],
            hisMsg['from'],
            formatDate(DateTime.now(), [HH, ':', nn, ':', ss]),
            hisMsg['session_id'],
        );
        _messages.insert(
            0,
            message);
      }
      setState(() {
        _messages = _messages;
      });
      client.on('broadcast', _messageProcess);
    }
  }


  void _messageProcess(rid, uid, info) {
    print('message: ' + info.toString());
    ChatMessage message = ChatMessage(
        randomNumeric(6),
        info['msg'],
        info['senderName'],
        uid,
        formatDate(DateTime.now(), [HH, ':', nn, ':', ss]),
        uid
    );

    _messages.insert(
        0,
        message);
    setState(() {
      _messages = _messages;
    });

  }

  @override
  void dispose() {
    print('Dispose chat widget!');

    _messages = <ChatMessage>[];

    super.dispose();
  }

//  testSend(){
//    var info1 =  {
//      "senderName":_displayName,
//      "msg": 'test message',
//    };
//
//    this.client.broadcast(_room,info1);
//  }

  void _handleSubmit(String text) {
    textEditingController.clear();

    if (text.length == 0 || text == '') {
      return;
    }

    var info =  {
      "senderName":_displayName,
      "msg": text,
    };

    this.client.broadcast(_room,info);

    var msg = ChatMessage(
        randomNumeric(6),
        text,
        this._displayName,
        this._displayName,
        formatDate(DateTime.now(), [HH, ':', nn, ':', ss]),
        'Me',
    );
    msg.isMe = true;
    _messages.insert(0, msg);
    setState(() {
      _messages = _messages;
    });
  }


  @override
  Widget build(BuildContext context) {

    return Scaffold(
      appBar: AppBar(
        title: Text('Text Message'),
      ),
      body: Stack(
        children: <Widget>[
          ListView.builder(
            padding: EdgeInsets.all(8.0),
            reverse: true,
            itemBuilder: (_, int index) => _messages[index],
            itemCount: _messages.length,
          ),
//          Positioned(
//            left: 0,
//            right: 0,
//            bottom: 0,
//            child: Container(
//              child: Row(
//                  children: <Widget>[
//                    TextField(
//                      controller: textEditingController,
//                      onSubmitted: _handleSubmit,
//                    ),
//                    Container(
//                      margin: const EdgeInsets.symmetric(horizontal: 8.0),
//                      child: IconButton(
//                        icon: Icon(Icons.send),
//                        onPressed: () => _handleSubmit(textEditingController.text),
//                      ),
//                    )
//                  ],
//            ),
//          ),
//          ),
        ],
      ),
    );
  }
}
