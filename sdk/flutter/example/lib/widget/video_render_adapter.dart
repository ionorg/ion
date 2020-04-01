import 'package:flutter_webrtc/webrtc.dart';
import 'package:flutter_ion/flutter_ion.dart';

class VideoRendererAdapter {
  String _mid;
  String _sid;
  bool _local;
  RTCVideoRenderer _renderer;
  Stream _stream;
  RTCVideoViewObjectFit _objectFit =
      RTCVideoViewObjectFit.RTCVideoViewObjectFitContain;
  VideoRendererAdapter(this._mid, this._stream, this._local, [this._sid]);

  setupSrcObject() async {
    if (_renderer == null) {
      _renderer = new RTCVideoRenderer();
      await _renderer.initialize();
    }
    _renderer.srcObject = _stream.stream;
    if (_local) {
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

  set objectFit(RTCVideoViewObjectFit objectFit) {
    _objectFit = objectFit;
    if (this._renderer != null) {
      _renderer.objectFit = _objectFit;
    }
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

  get mid => _mid;

  get sid => _sid;

  get renderer => _renderer;

  get stream => _stream.stream;
}
