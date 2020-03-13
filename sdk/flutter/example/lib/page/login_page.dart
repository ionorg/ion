import 'package:flutter/material.dart';
import 'package:pion_sfu_example/helper/ion_helper.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../utils/utils.dart';

class LoginPage extends StatefulWidget {
  final IonHelper _helper;
  LoginPage(this._helper, {Key key, this.title}) : super(key: key);
  final String title;
  @override
  _LoginPageState createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  String _server;
  String _roomID;
  SharedPreferences prefs;

  @override
  initState() {
    super.initState();
    init();
  }

  init() async {
    IonHelper helper = widget._helper;
    prefs = await SharedPreferences.getInstance();
    setState(() {
      _server = prefs.getString('server') ?? 'pionion.org';
      _roomID = prefs.getString('room') ?? 'room1';
    });

    helper.on('transport-open', () {
      Navigator.pushNamed(context, '/meeting');
    });
  }

  handleJoin() async {
    IonHelper helper = widget._helper;
    prefs.setString('server', _server);
    prefs.setString('room', _roomID);
    prefs.commit();
    helper.connect(_server);
  }

  Widget buildJoinView(context) {
    return Align(
        alignment: Alignment(0, 0),
        child: Column(
            crossAxisAlignment: CrossAxisAlignment.center,
            mainAxisAlignment: MainAxisAlignment.center,
            children: <Widget>[
              SizedBox(
                  width: 260.0,
                  child: TextField(
                      keyboardType: TextInputType.text,
                      textAlign: TextAlign.center,
                      decoration: InputDecoration(
                        contentPadding: EdgeInsets.all(10.0),
                        border: UnderlineInputBorder(
                            borderSide: BorderSide(color: Colors.black12)),
                        hintText: _server ?? 'Enter Ion Server.',
                      ),
                      onChanged: (value) {
                        setState(() {
                          _server = value;
                        });
                      },
                      controller:
                          TextEditingController.fromValue(TextEditingValue(
                        text: '${this._server == null ? "" : this._server}',
                        selection: TextSelection.fromPosition(TextPosition(
                            affinity: TextAffinity.downstream,
                            offset: '${this._server}'.length)),
                      )))),
              SizedBox(
                  width: 260.0,
                  child: TextField(
                      keyboardType: TextInputType.text,
                      textAlign: TextAlign.center,
                      decoration: InputDecoration(
                        contentPadding: EdgeInsets.all(10.0),
                        border: UnderlineInputBorder(
                            borderSide: BorderSide(color: Colors.black12)),
                        hintText: _roomID ?? 'Enter RoomID.',
                      ),
                      onChanged: (value) {
                        setState(() {
                          _roomID = value;
                        });
                      },
                      controller:
                          TextEditingController.fromValue(TextEditingValue(
                        text: '${this._roomID == null ? "" : this._roomID}',
                        selection: TextSelection.fromPosition(TextPosition(
                            affinity: TextAffinity.downstream,
                            offset: '${this._roomID}'.length)),
                      )))),
              SizedBox(width: 260.0, height: 48.0),
              InkWell(
                child: Container(
                  width: 220.0,
                  height: 48.0,
                  decoration: BoxDecoration(
                    border: Border.all(
                      color: string2Color('#e13b3f'),
                      width: 1,
                    ),
                  ),
                  child: Center(
                    child: Text(
                      'Join',
                      style: TextStyle(
                        fontSize: 16.0,
                        color: Colors.black,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ),
                onTap: () {
                  if (_roomID != null) {
                    handleJoin();
                    prefs.setString('room', _roomID);
                    return;
                  }
                  showDialog<Null>(
                    context: context,
                    barrierDismissible: false,
                    builder: (BuildContext context) {
                      return AlertDialog(
                        title: Text('Client id is empty'),
                        content: Text('Please enter Ion room id!'),
                        actions: <Widget>[
                          FlatButton(
                            child: Text('Ok'),
                            onPressed: () {
                              Navigator.of(context).pop();
                            },
                          ),
                        ],
                      );
                    },
                  );
                },
              ),
            ]));
  }

  @override
  Widget build(BuildContext context) {
    return OrientationBuilder(builder: (context, orientation) {
      return Scaffold(
          appBar: orientation == Orientation.portrait
              ? AppBar(
                  title: Text('PION'),
                )
              : null,
          body: Stack(children: <Widget>[
            Center(child: buildJoinView(context)),
            Positioned(
              bottom: 6.0,
              right: 6.0,
              child: FlatButton(
                onPressed: () {
                  Navigator.pushNamed(context, '/settings');
                },
                child: Text(
                  "Settings",
                  style: TextStyle(fontSize: 16.0, color: Colors.black54),
                ),
              ),
            ),
          ]));
    });
  }
}
