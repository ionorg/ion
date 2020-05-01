import 'package:flutter/material.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'package:flutter_icons/flutter_icons.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../widget/video_render_adapter.dart';
import '../helper/ion_helper.dart';
import './chat_page.dart';

class MeetingPage extends StatefulWidget {
  final IonHelper _helper;
  MeetingPage(this._helper, {Key key}) : super(key: key);
  @override
  _MeetingPageState createState() => _MeetingPageState();
}

class _MeetingPageState extends State<MeetingPage> {
  SharedPreferences prefs;
  List<VideoRendererAdapter> _remoteVideos = List();
  VideoRendererAdapter _localVideo;

  bool _cameraOff = false;
  bool _microphoneOff = false;
  bool _speakerOn = true;
  var _scaffoldkey = new GlobalKey<ScaffoldState>();
  var _messages = [];
  var name;
  var room;

  final double LOCAL_VIDEO_WIDTH = 114.0;
  final double LOCAL_VIDEO_HEIGHT = 72.0;

  @override
  initState() {
    super.initState();
    init();
  }

  init() async {
    prefs = await SharedPreferences.getInstance();
    var helper = widget._helper;
    var client = widget._helper.client;

    client.on('peer-join', (rid, id, info) async {
      var name = info['name'];
      this._showSnackBar(":::Peer [$id:$name] join:::");
    });

    client.on('peer-leave', (rid, id) async {
      this._showSnackBar(":::Peer [$id] leave:::");
    });

    client.on('stream-add', (rid, mid, info, tracks) async {
      var bandwidth = prefs.getString('bandwidth') ?? '512';
      var stream = await client.subscribe(rid, mid, tracks, bandwidth);
      var adapter = VideoRendererAdapter(stream.mid, stream, false, mid);
      await adapter.setupSrcObject();
      this.setState(() {
        _remoteVideos.add(adapter);
      });
      this._showSnackBar(":::stream-add [$mid]:::");
    });

    client.on('stream-remove', (rid, mid) async {
      var adapter = _remoteVideos.firstWhere((item) => item.sid == mid);
      if (adapter != null) {
        await adapter.dispose();
        this.setState(() {
          _remoteVideos.remove(adapter);
        });
      }
      this._showSnackBar(":::stream-remove [$mid]:::");
    });


    name = prefs.getString('display_name') ?? 'Guest';
    room = prefs.getString('room') ?? 'room1';

    helper.join(room, name);
    try {
      var resolution = prefs.getString('resolution') ?? 'vga';
      var bandwidth = prefs.getString('bandwidth') ?? '512';
      var codec = prefs.getString('codec') ?? 'vp8';
      client
          .publish(true, true, false, codec, bandwidth, resolution)
          .then((stream) async {
        var adapter = VideoRendererAdapter(stream.mid, stream, true);
        await adapter.setupSrcObject();
        var localStream = stream.stream;
        MediaStreamTrack audioTrack = localStream.getAudioTracks()[0];
        audioTrack.enableSpeakerphone(true);
        this.setState(() {
          _localVideo = adapter;
        });
      });
    } catch (error) {}
  }

  _cleanUp() async {
    var helper = widget._helper;
    var rid = helper.roomId;
    var client = helper.client;

    if (_localVideo != null) {
      // stop local video
      var stream = _localVideo.stream;
      await client.unpublish(_localVideo.mid);
      await stream.dispose();
      _localVideo = null;
    }

    _remoteVideos.forEach((item) async {
      var stream = item.stream;
      try {
        await client.unsubscribe(rid, item.mid);
        await stream.dispose();
      } catch (error) {}
    });
    this.setState(() {});
    _remoteVideos.clear();
    await widget._helper.close();
    Navigator.of(context).pop();
  }

  Widget buildVideoView(VideoRendererAdapter adapter) {
    return Container(
      alignment: Alignment.center,
      child: RTCVideoView(adapter.renderer),
      color: Colors.black,
    );
  }

  Widget _buildMainVideo() {
    if (_remoteVideos.length == 0)
      return Image.asset(
        'assets/images/loading.jpeg',
        fit: BoxFit.cover,
      );

    var adapter = _remoteVideos[0];
    return GestureDetector(
        onDoubleTap: () {
          adapter.switchObjFit();
        },
        child: RTCVideoView(adapter.renderer));
  }

  Widget _buildLocalVideo(Orientation orientation) {
    if (_localVideo != null) {
      return SizedBox(
          width: (orientation == Orientation.portrait)
              ? LOCAL_VIDEO_HEIGHT
              : LOCAL_VIDEO_WIDTH,
          height: (orientation == Orientation.portrait)
              ? LOCAL_VIDEO_WIDTH
              : LOCAL_VIDEO_HEIGHT,
          child: Container(
            decoration: BoxDecoration(
              color: Colors.black87,
              border: Border.all(
                color: Colors.white,
                width: 0.5,
              ),
            ),
            child: GestureDetector(
                onTap: () {
                  _switchCamera();
                },
                onDoubleTap: () {
                  _localVideo.switchObjFit();
                },
                child: RTCVideoView(_localVideo.renderer)),
          ));
    }
    return Container();
  }

  List<Widget> _buildVideoViews() {
    List<Widget> views = new List<Widget>();
    if (_remoteVideos.length > 1)
      _remoteVideos.getRange(1, _remoteVideos.length).forEach((adapter) {
        adapter.objectFit = RTCVideoViewObjectFit.RTCVideoViewObjectFitCover;
        views.add(_buildVideo(adapter));
      });
    return views;
  }

  _swapVideoPostion(adapter) {
    var index =
        _remoteVideos.indexWhere((element) => element.mid == adapter.mid);
    if (index == -1) return;
    setState(() {
      var temp = _remoteVideos[0];
      _remoteVideos[0] = _remoteVideos[index];
      _remoteVideos[index] = temp;
    });
  }

  Widget _buildVideo(VideoRendererAdapter adapter) {
    return SizedBox(
      width: 120,
      height: 90,
      child: Container(
        decoration: BoxDecoration(
          color: Colors.black87,
          border: Border.all(
            color: Colors.white,
            width: 1.0,
          ),
        ),
        child: GestureDetector(
            onTap: () => _swapVideoPostion(adapter),
            onDoubleTap: () => adapter.switchObjFit(),
            child: RTCVideoView(adapter.renderer)),
      ),
    );
  }

  //Switch speaker/earpiece
  _switchSpeaker() {
    this.setState(() {
      _speakerOn = !_speakerOn;
      MediaStreamTrack audioTrack = _localVideo.stream.getAudioTracks()[0];
      audioTrack.enableSpeakerphone(_speakerOn);
      _showSnackBar(
          ":::Switch to " + (_speakerOn ? "speaker" : "earpiece") + ":::");
    });
  }

  //Switch local camera
  _switchCamera() {
    if (_localVideo != null && _localVideo.stream.getVideoTracks().length > 0) {
      _localVideo.stream.getVideoTracks()[0].switchCamera();
    } else {
      _showSnackBar(":::Unable to switch the camera:::");
    }
  }

  //Open or close local video
  _turnCamera() {
    if (_localVideo != null && _localVideo.stream.getVideoTracks().length > 0) {
      var muted = !_cameraOff;
      setState(() {
        _cameraOff = muted;
      });
      _localVideo.stream.getVideoTracks()[0].enabled = !muted;
    } else {
      _showSnackBar(":::Unable to operate the camera:::");
    }
  }

  //Open or close local audio
  _turnMicrophone() {
    if (_localVideo != null && _localVideo.stream.getAudioTracks().length > 0) {
      var muted = !_microphoneOff;
      setState(() {
        _microphoneOff = muted;
      });
      _localVideo.stream.getAudioTracks()[0].enabled = !muted;

      if (muted) {
        _showSnackBar(":::The microphone is muted:::");
      } else {
        _showSnackBar(":::The microphone is unmuted:::");
      }
    } else {}
  }

  //Leave current video room
  _hangUp() {
    showDialog(
        context: context,
        builder: (_) => AlertDialog(
                title: Text("Hangup"),
                content: Text("Are you sure to leave the room?"),
                actions: <Widget>[
                  FlatButton(
                    child: Text("Cancel"),
                    onPressed: () {
                      Navigator.of(context).pop();
                    },
                  ),
                  FlatButton(
                    child: Text(
                      "Hangup",
                      style: TextStyle(color: Colors.red),
                    ),
                    onPressed: () {
                      Navigator.of(context).pop();
                      _cleanUp();
                    },
                  )
                ]));
  }

  _showSnackBar(String message) {
    final snackBar = SnackBar(
      content: Text(
        message,
        style: TextStyle(color: Colors.white),
      ),
      duration: Duration(
        milliseconds: 1000,
      ),
    );
    _scaffoldkey.currentState.showSnackBar(snackBar);
  }

  Widget _buildLoading() {
    return Center(
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[
          Center(
            child: CircularProgressIndicator(
              valueColor: AlwaysStoppedAnimation(Colors.white),
            ),
          ),
          SizedBox(
            width: 10,
          ),
          Text(
            'Waiting for others to join...',
            style: TextStyle(
                color: Colors.white,
                fontSize: 22.0,
                fontWeight: FontWeight.bold),
          ),
        ],
      ),
    );
  }

  //tools
  List<Widget> _buildTools() {
    return <Widget>[
      SizedBox(
        width: 36,
        height: 36,
        child: RawMaterialButton(
          shape: CircleBorder(
            side: BorderSide(
              color: Colors.white,
              width: 1,
            ),
          ),
          child: Icon(
            MaterialCommunityIcons.getIconData(
                _cameraOff ? "video-off" : "video"),
            color: _cameraOff ? Colors.red : Colors.white,
          ),
          onPressed: _turnCamera,
        ),
      ),
      SizedBox(
        width: 36,
        height: 36,
        child: RawMaterialButton(
          shape: CircleBorder(
            side: BorderSide(
              color: Colors.white,
              width: 1,
            ),
          ),
          child: Icon(
            MaterialCommunityIcons.getIconData("video-switch"),
            color: Colors.white,
          ),
          onPressed: _switchCamera,
        ),
      ),
      SizedBox(
        width: 36,
        height: 36,
        child: RawMaterialButton(
          shape: CircleBorder(
            side: BorderSide(
              color: Colors.white,
              width: 1,
            ),
          ),
          child: Icon(
            MaterialCommunityIcons.getIconData(
                _microphoneOff ? "microphone-off" : "microphone"),
            color: _microphoneOff ? Colors.red : Colors.white,
          ),
          onPressed: _turnMicrophone,
        ),
      ),
      SizedBox(
        width: 36,
        height: 36,
        child: RawMaterialButton(
          shape: CircleBorder(
            side: BorderSide(
              color: Colors.white,
              width: 1,
            ),
          ),
          child: Icon(
            MaterialIcons.getIconData(
                _speakerOn ? "volume-up" : "speaker-phone"),
            color: Colors.white,
          ),
          onPressed: _switchSpeaker,
        ),
      ),
      SizedBox(
        width: 36,
        height: 36,
        child: RawMaterialButton(
          shape: CircleBorder(
            side: BorderSide(
              color: Colors.white,
              width: 1,
            ),
          ),
          child: Icon(
            MaterialCommunityIcons.getIconData("phone-hangup"),
            color: Colors.red,
          ),
          onPressed: _hangUp,
        ),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) {
    return OrientationBuilder(builder: (context, orientation) {
      return SafeArea(
        child: Scaffold(
          key: _scaffoldkey,
          body: orientation == Orientation.portrait
              ? Container(
                  color: Colors.black87,
                  child: Stack(
                    children: <Widget>[
                      Positioned(
                        left: 0,
                        right: 0,
                        top: 0,
                        bottom: 0,
                        child: Container(
                          color: Colors.black54,
                          child: Stack(
                            children: <Widget>[
                              Positioned(
                                left: 0,
                                right: 0,
                                top: 0,
                                bottom: 0,
                                child: Container(
                                  child: _buildMainVideo(),
                                ),
                              ),
                              Positioned(
                                right: 10,
                                top: 48,
                                child: Container(
                                  child: _buildLocalVideo(orientation),
                                ),
                              ),
                              Positioned(
                                left: 0,
                                right: 0,
                                bottom: 48,
                                height: 90,
                                child: Container(
                                  margin: EdgeInsets.all(6.0),
                                  child: ListView(
                                    scrollDirection: Axis.horizontal,
                                    children: _buildVideoViews(),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                      (_remoteVideos.length == 0)
                          ? _buildLoading()
                          : Container(),
                      Positioned(
                        left: 0,
                        right: 0,
                        bottom: 0,
                        height: 48,
                        child: Stack(
                          children: <Widget>[
                            Opacity(
                              opacity: 0.5,
                              child: Container(
                                color: Colors.black,
                              ),
                            ),
                            Container(
                              height: 48,
                              margin: EdgeInsets.all(0.0),
                              child: Row(
                                mainAxisAlignment:
                                    MainAxisAlignment.spaceEvenly,
                                crossAxisAlignment: CrossAxisAlignment.center,
                                children: _buildTools(),
                              ),
                            ),
                          ],
                        ),
                      ),
                      Positioned(
                        left: 0,
                        right: 0,
                        top: 0,
                        height: 48,
                        child: Stack(
                          children: <Widget>[
                            Opacity(
                              opacity: 0.5,
                              child: Container(
                                color: Colors.black,
                              ),
                            ),
                            Container(
                              margin: EdgeInsets.all(0.0),
                              child: Center(
                                child: Text(
                                  'Ion Flutter Demo',
                                  style: TextStyle(
                                    color: Colors.white,
                                    fontSize: 18.0,
                                  ),
                                ),
                              ),
                            ),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              crossAxisAlignment: CrossAxisAlignment.center,
                              children: <Widget>[
                                IconButton(
                                  icon: Icon(
                                    Icons.people,
                                    size: 28.0,
                                    color: Colors.white,
                                  ),
                                ),
                                //Chat message
                                IconButton(
                                  icon: Icon(
                                    Icons.chat_bubble_outline,
                                    size: 28.0,
                                    color: Colors.white,
                                  ),
                                  onPressed: () {
                                    Navigator.push(
                                      context,
                                      MaterialPageRoute(
                                        builder: (context) => ChatPage(widget._helper.client,
                                            this._messages, this.name,this.room),
                                      ),
                                    );
                                  },
                                ),
                              ],
                            ),

                          ],
                        ),
                      ),
                    ],
                  ),
                )
              : Container(
                  color: Colors.black54,
                  child: Stack(
                    children: <Widget>[
                      Positioned(
                        left: 0,
                        right: 0,
                        top: 0,
                        bottom: 0,
                        child: Container(
                          color: Colors.black87,
                          child: Stack(
                            children: <Widget>[
                              Positioned(
                                left: 0,
                                right: 0,
                                top: 0,
                                bottom: 0,
                                child: Container(
                                  child: _buildMainVideo(),
                                ),
                              ),
                              Positioned(
                                right: 60,
                                top: 10,
                                child: Container(
                                  child: _buildLocalVideo(orientation),
                                ),
                              ),
                              Positioned(
                                left: 0,
                                top: 0,
                                bottom: 0,
                                width: 120,
                                child: Container(
                                  margin: EdgeInsets.all(6.0),
                                  child: ListView(
                                    scrollDirection: Axis.vertical,
                                    children: _buildVideoViews(),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                      (_remoteVideos.length == 0)
                          ? _buildLoading()
                          : Container(),
                      Positioned(
                        top: 0,
                        right: 0,
                        bottom: 0,
                        width: 48,
                        child: Stack(
                          children: <Widget>[
                            Opacity(
                              opacity: 0.5,
                              child: Container(
                                color: Colors.black,
                              ),
                            ),
                            Container(
                              width: 48,
                              margin: EdgeInsets.all(0.0),
                              child: Column(
                                mainAxisSize: MainAxisSize.max,
                                mainAxisAlignment:
                                    MainAxisAlignment.spaceEvenly,
                                crossAxisAlignment: CrossAxisAlignment.center,
                                children: _buildTools(),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
        ),
      );
    });
  }
}
