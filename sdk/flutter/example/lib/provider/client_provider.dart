import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_ion/flutter_ion.dart';
import 'package:flutter_webrtc/webrtc.dart';
import '../widget/video_render_adapter.dart';

class ClientProvider with ChangeNotifier {
  Client _client;
  bool _inCalling = false;
  bool _connected = false;
  String roomId;
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
      var adapter = VideoRendererAdapter(stream.mid, true);
      await adapter.setSrcObject(stream.stream);
      _localVideoAdapter = adapter;
      notifyListeners();
    } catch (error) {}
  }

  cleanUp() async {
    _videoRendererAdapters.forEach((item) async {
      if (item.local) {
        await item.stream.dispose();
        if(_client != null){
          await _client.unpublish(item.id);
        }
      } else {
        await item.stream.dispose();
        await _client.unsubscribe(this.roomId, item.id);
      }
    });
    _videoRendererAdapters.clear();
    if (_client != null) {
      await _client.leave();
      _client.close();
      _client = null;
    }
  }
}
