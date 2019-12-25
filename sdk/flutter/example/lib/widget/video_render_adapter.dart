import 'package:flutter_webrtc/webrtc.dart';

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