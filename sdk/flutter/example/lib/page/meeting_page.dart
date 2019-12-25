import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:provider/provider.dart';
import 'package:flutter_webrtc/webrtc.dart';
import '../widget/video_render_adapter.dart';
import '../provider/client_provider.dart';
import '../router/application.dart';

class MeetingPage extends StatelessWidget {
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

class _MyHomePageState extends State<MyHomePage> {
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

  handleLeave() async{
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

  List<Widget> _buildVideoViews() {
    List<Widget> views = List<Widget>();
    _videoRendererAdapters.forEach((adapter) {
      views.add(buildVideoView(adapter));
    });
    return views;
  }

  Widget buildStreamsGridView() {
    return GridView.extent(
      maxCrossAxisExtent: 300.0,
      padding: const EdgeInsets.all(1.0),
      mainAxisSpacing: 1.0,
      crossAxisSpacing: 1.0,
      children: _buildVideoViews(),
    );
  }

  @override
  Widget build(BuildContext context) {
    _inCalling = Provider.of<ClientProvider>(context).inCalling;
    return OrientationBuilder(builder: (context, orientation) {
      return Scaffold(
        appBar: orientation == Orientation.portrait
            ? AppBar(
          title: Text(widget.title),
        )
            : null,
        body: Center(
          child: Consumer<ClientProvider>(builder: (BuildContext context, ClientProvider clientProvider, Widget child) {
            //获取消息列表数据
            _videoRendererAdapters = clientProvider.videoRendererAdapters;
            return buildStreamsGridView();
          }),
        ),
        floatingActionButton: _inCalling
            ? FloatingActionButton(
          onPressed: (){
            handleLeave();
          },
          backgroundColor: Colors.red,
          tooltip: 'Increment',
          child: Icon(Icons.call_end),
        )
            : null,
      );
    });
  }
}
