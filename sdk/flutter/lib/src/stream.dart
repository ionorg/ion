import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'logger.dart' show Logger;

// 'vga': {'minWidth': '640', 'minHeight': '360'},
final Map<String, dynamic> resolutions = {
  'qvga': {'width': '320', 'height': '180'},
  'vga': {'width': '640', 'height': '360'},
  'shd': {'width': '960', 'height': '540'},
  'hd': {'width': '1280', 'height': '720'},
  'fullhd': {'width': '1920', 'height': '1080'}
};

class Stream extends EventEmitter {
  var logger = new Logger("Ion::Stream");
  String _mid;
  MediaStream _stream;
  Stream([this._mid, this._stream]);

  init(
      [sender = false,
      audio = true,
      video = true,
      screen = false,
      quality = 'hd']) async {
    if (sender) {
      if (screen) {
        this._stream = await navigator
            .getDisplayMedia(_buildMediaConstraints(false, true));
      } else {
        Map<String, dynamic> videoConstrains = {
          "mandatory": {
            "minWidth": resolutions[quality]['width'],
            "minHeight": resolutions[quality]['height'],
            "minFrameRate": '30',
          },
          "facingMode": 'user',
        };
        this._stream = await navigator
            .getUserMedia(_buildMediaConstraints(audio, videoConstrains));
      }
    }
  }

  set mid(id) {
    this._mid = id;
  }

  String get mid => _mid;

  MediaStream get stream => _stream;

  _buildMediaConstraints(audio, video) {
    return {"audio": audio, "video": video ?? false};
  }
}
