import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_ion/flutter_ion.dart';
import '../widget/video_render_adapter.dart';

class ClientProvider with ChangeNotifier {
  Client _client;
  bool _inCalling = false;
  bool _connected = false;
  String roomId;
  Map<Stream, bool> _streams = Map();
  List<VideoRendererAdapter> _videoRendererAdapters = List();
  VideoRendererAdapter _localVideoAdapter;

  get videoRendererAdapters => _videoRendererAdapters;
  get localVideoAdapter => _localVideoAdapter;
  get connected => _connected;
  get inCalling => _inCalling;
  get client => _client;

  connect(host) async {
    if (_client == null) {
      var url = 'https://$host:8443/ws';
      _client = Client(url);

      _client.on('transport-open', () {
        print('transport-open');
        _connected = true;
      });

      _client.on('transport-closed', () {
        print('transport-closed');
        _connected = false;
      });

      _client.on('stream-add', (rid, mid, info) async {
        var stream = await _client.subscribe(rid, mid);
        _streams[stream] = false;
        var adapter = VideoRendererAdapter(stream.mid, false);
        await adapter.setSrcObject(stream.stream);
        _videoRendererAdapters.add(adapter);
        notifyListeners();
      });

      _client.on('stream-remove', (rid, mid) async {
        var adapter =
            _videoRendererAdapters.firstWhere((item) => item.id == mid);
        await adapter.dispose();
        _videoRendererAdapters.remove(adapter);
        notifyListeners();
      });

      _client.on('peer-join', (rid, id, info) async {});

      _client.on('peer-leave', (rid, id) async {});
    }

    await _client.connect();
  }

  join(String roomId) async {
    try {
      roomId = roomId;
      await _client.join(roomId, {'name': 'Guest'});
      _inCalling = true;
      var stream = await _client.publish();
      _streams[stream] = true;
      var adapter = VideoRendererAdapter(stream.mid, true);
      await adapter.setSrcObject(stream.stream);
      _localVideoAdapter = adapter;
      notifyListeners();
    } catch (error) {}
  }

  cleanUp() async {

    _streams.forEach((stream, local) {
      stream.stream.dispose();
      if(local){
        _client.unpublish(stream.mid);
      } else {
        _client.unsubscribe(roomId, stream.mid);
      }
    });

    if (_client != null) {
      await _client.leave();
      _client.close();
      _client = null;
    }
    _videoRendererAdapters.clear();
  }
}
