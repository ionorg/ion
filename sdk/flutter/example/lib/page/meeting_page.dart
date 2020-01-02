import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:flutter_webrtc/webrtc.dart';
import '../widget/video_render_adapter.dart';
import '../provider/client_provider.dart';
import '../router/application.dart';

class MeetingPage extends StatefulWidget {
  @override
  _MeetingPageState createState() => _MeetingPageState();
}

class _MeetingPageState extends State<MeetingPage> {
  SharedPreferences prefs;
  bool _inCalling = false;
  List<VideoRendererAdapter> _videoRendererAdapters = List();

  @override
  initState() {
    super.initState();
    init();
  }

  init() async {
    prefs = await SharedPreferences.getInstance();
  }

  handleLeave() async {
    await Provider.of<ClientProvider>(context).cleanUp();
    Application.router.navigateTo(context, "/login");
  }

  Widget buildVideoView(VideoRendererAdapter adapter) {
    return Container(
      alignment: Alignment.center,
      child: RTCVideoView(adapter.renderer),
      color: Colors.black,
    );
  }

  Widget _buildMainVideo() {
    if (_videoRendererAdapters.length == 0)
      return Image.asset(
        'assets/images/loading.jpeg',
        fit: BoxFit.cover,
      );

    var adapter = _videoRendererAdapters[0];
    return GestureDetector(
        onDoubleTap: () {
          adapter.switchObjFit();
        },
        child: RTCVideoView(adapter.renderer));
  }

  List<Widget> _buildVideoViews() {
    List<Widget> views = new List<Widget>();
    if (_videoRendererAdapters.length > 1)
      _videoRendererAdapters
          .getRange(1, _videoRendererAdapters.length)
          .forEach((adapter) {
        views.add(_buildVideo(adapter));
      });
    return views;
  }

  swapVideoPostion(int x, int y) {
    var src = _videoRendererAdapters[x];
    var dest = _videoRendererAdapters[y];
    var srcStream = src.stream;
    src.setSrcObject(null);
    dest.setSrcObject(srcStream);
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
            onTap: () async {
              var mainVideoAdapter = _videoRendererAdapters[0];
              var mainStream = mainVideoAdapter.stream;
              await mainVideoAdapter.setSrcObject(adapter.stream);
              await adapter.setSrcObject(mainStream);
            },
            onDoubleTap: () {
              adapter.switchObjFit();
            },
            child: RTCVideoView(adapter.renderer)),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    _inCalling = Provider.of<ClientProvider>(context).inCalling;
    return OrientationBuilder(builder: (context, orientation) {
      return Scaffold(
        appBar: orientation == Orientation.portrait
            ? AppBar(
                title: Text('Ion Flutter Demo'),
                centerTitle: true,
                automaticallyImplyLeading: false,
              )
            : null,
        body: Consumer<ClientProvider>(builder: (BuildContext context,
            ClientProvider clientProvider, Widget child) {
          _videoRendererAdapters = clientProvider.videoRendererAdapters;
          return orientation == Orientation.portrait
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
                                left: 0,
                                right: 0,
                                bottom: 48,
                                height: 90,
                                child: ListView(
                                  scrollDirection: Axis.horizontal,
                                  children: _buildVideoViews(),
                                ),
                              ),
                            ],
                          ),
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
                                  )),
                              Positioned(
                                left: 0,
                                top: 0,
                                bottom: 0,
                                width: 120,
                                child: ListView(
                                  scrollDirection: Axis.vertical,
                                  children: _buildVideoViews(),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ],
                  ),
                );
        }),
      );
    });
  }
}
