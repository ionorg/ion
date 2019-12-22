import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter_ion/flutter_ion.dart';
import 'package:flutter_webrtc/webrtc.dart';

void main() => runApp(MyApp());

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Ion Flutter Demo',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: MyHomePage(title: 'Ion Flutter Demo'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  MyHomePage({Key key, this.title}) : super(key: key);
  final String title;
  @override
  _MyHomePageState createState() => _MyHomePageState();
}

class VideoRendererAdapter {
  String _id;
  bool _local;
  RTCVideoRenderer _renderer;
  MediaStream _stream;
  RTCVideoViewObjectFit _objectFit =
      RTCVideoViewObjectFit.RTCVideoViewObjectFitContain;
  VideoRendererAdapter(this._id, this._local);

  setSrcObject(MediaStream stream, {bool localVideo = false}) async {
    if (_renderer == null) {
      _renderer = new RTCVideoRenderer();
      await _renderer.initialize();
    }
    _stream = stream;
    _renderer.srcObject = _stream;
    if (localVideo) {
      _objectFit = RTCVideoViewObjectFit.RTCVideoViewObjectFitCover;
      _renderer.mirror = true;
      _renderer.objectFit = _objectFit;
    }
  }

  switchObjFit() {
    _objectFit =
        (_objectFit == RTCVideoViewObjectFit.RTCVideoViewObjectFitContain)
            ? RTCVideoViewObjectFit.RTCVideoViewObjectFitCover
            : RTCVideoViewObjectFit.RTCVideoViewObjectFitContain;
    _renderer.objectFit = _objectFit;
  }

  dispose() async {
    if (_renderer != null) {
      print('dispose for texture id ' + _renderer.textureId.toString());
      _renderer.srcObject = null;
      await _renderer.dispose();
      _renderer = null;
    }
  }

  get local => _local;

  get id => _id;

  get renderer => _renderer;

  get stream => _stream;

  get streamId => _stream.id;
}

class _MyHomePageState extends State<MyHomePage> {
  String _server;
  String _roomID;
  SharedPreferences prefs;
  Client _client;
  bool _inCalling = false;
  bool _connected = false;
  List<VideoRendererAdapter> _videoRendererAdapters = new List();

  @override
  initState() {
    super.initState();
    init();
  }

  init() async {
    prefs = await SharedPreferences.getInstance();
    setState(() {
      _server = prefs.getString('server') ?? 'pionion.org';
      _roomID = prefs.getString('room') ?? 'room1';
    });
  }

  _cleanUp() async {
    _videoRendererAdapters.forEach((item) async {
      if (item.local) {
        await item.stream.dispose();
        await _client.unpublish(item.id);
      } else {
        await item.stream.dispose();
        await _client.unsubscribe(this._roomID, item.id);
      }
    });
    _videoRendererAdapters.clear();
    if (_client != null) {
      await _client.leave();
      _client.close();
      _client = null;
    }
    this.setState(() {});
  }

  handleConnect() async {
    if (_client == null) {
      var url = 'https://' + _server + ':8443/ws';
      _client = new Client(url);
      _client.on('transport-open', () {
        print('transport-open');
        setState(() {
          _connected = true;
        });
      });
      _client.on('transport-closed', () {
        print('transport-closed');
        setState(() {
          _connected = false;
        });
      });

      _client.on('stream-add', (rid, mid, info) async {
        var stream = await _client.subscribe(rid, mid);
        var adapter = new VideoRendererAdapter(stream.mid, false);
        await adapter.setSrcObject(stream.stream);
        setState(() {
          _videoRendererAdapters.add(adapter);
        });
      });

      _client.on('stream-remove', (rid, mid) async {
        var adapter =
            _videoRendererAdapters.firstWhere((item) => item.id == mid);
        await adapter.dispose();
        setState(() {
          _videoRendererAdapters.remove(adapter);
        });
      });

      _client.on('peer-join', (rid, id, info) async {});

      _client.on('peer-leave', (rid, id) async {});
    }
  }

  handleJoin() async {
    try {
      await _client.join(_roomID, {'name': 'Guest'});
      setState(() {
        _inCalling = true;
      });
      var stream = await _client.publish();
      var adapter = new VideoRendererAdapter(stream.mid, true);
      await adapter.setSrcObject(stream.stream);
      setState(() {
        _videoRendererAdapters.add(adapter);
      });
    } catch (error) {}
  }

  handleLeave() async {
    setState(() {
      _inCalling = false;
    });
    await _cleanUp();
  }

  Widget buildJoinView(context) {
    return new Align(
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
              SizedBox(
                  width: 220.0,
                  height: 48.0,
                  child: MaterialButton(
                    child: Text(
                      'Join',
                      style: TextStyle(fontSize: 16.0, color: Colors.white),
                    ),
                    color: Colors.blue,
                    textColor: Colors.white,
                    onPressed: () {
                      if (_roomID != null) {
                        handleJoin();
                        prefs.setString('room', _roomID);
                        return;
                      }
                      showDialog<Null>(
                        context: context,
                        barrierDismissible: false,
                        builder: (BuildContext context) {
                          return new AlertDialog(
                            title: new Text('Client id is empty'),
                            content: new Text('Please enter Ion room id!'),
                            actions: <Widget>[
                              new FlatButton(
                                child: new Text('Ok'),
                                onPressed: () {
                                  Navigator.of(context).pop();
                                },
                              ),
                            ],
                          );
                        },
                      );
                    },
                  ))
            ]));
  }

  Widget buildConnectView(context) {
    return new Align(
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
                        hintText: _server ?? 'Enter Ion server.',
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
              SizedBox(width: 260.0, height: 48.0),
              SizedBox(
                  width: 220.0,
                  height: 48.0,
                  child: MaterialButton(
                    child: Text(
                      'Connect',
                      style: TextStyle(fontSize: 16.0, color: Colors.white),
                    ),
                    color: Colors.blue,
                    textColor: Colors.white,
                    onPressed: () {
                      if (_server != null) {
                        handleConnect();
                        prefs.setString('server', _server);
                        return;
                      }
                      showDialog<Null>(
                        context: context,
                        barrierDismissible: false,
                        builder: (BuildContext context) {
                          return new AlertDialog(
                            title: new Text('Server is empty'),
                            content:
                                new Text('Please enter Pion-Client address!'),
                            actions: <Widget>[
                              new FlatButton(
                                child: new Text('Ok'),
                                onPressed: () {
                                  Navigator.of(context).pop();
                                },
                              ),
                            ],
                          );
                        },
                      );
                    },
                  ))
            ]));
  }

  Widget buildVideoView(VideoRendererAdapter adapter) {
    return Container(
      alignment: Alignment.center,
      child: RTCVideoView(adapter.renderer),
      color: Colors.black,
    );
  }

  List<Widget> _buildVideoViews() {
    List<Widget> views = new List<Widget>();
    _videoRendererAdapters.forEach((adapter) {
      views.add(buildVideoView(adapter));
    });
    return views;
  }

  Widget buildStreamsGridView() {
    return new GridView.extent(
      maxCrossAxisExtent: 300.0,
      padding: const EdgeInsets.all(1.0),
      mainAxisSpacing: 1.0,
      crossAxisSpacing: 1.0,
      children: _buildVideoViews(),
    );
  }

  @override
  Widget build(BuildContext context) {
    return OrientationBuilder(builder: (context, orientation) {
      return Scaffold(
        appBar: orientation == Orientation.portrait
            ? AppBar(
                title: Text(widget.title),
              )
            : null,
        body: Center(
          child: _connected
              ? _inCalling ? buildStreamsGridView() : buildJoinView(context)
              : buildConnectView(context),
        ),
        floatingActionButton: _inCalling
            ? FloatingActionButton(
                onPressed: handleLeave,
                backgroundColor: Colors.red,
                tooltip: 'Increment',
                child: Icon(Icons.call_end),
              )
            : null,
      );
    });
  }
}
