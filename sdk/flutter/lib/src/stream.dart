import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'logger.dart' show Logger;

class Stream extends EventEmitter {
  var logger = new Logger("Ion::Stream");
  String _mid;
  MediaStream _stream;
  Stream([this._mid, this._stream]);

  init([sender = false, audio = true, video = true, screen = false]) async {
    if (sender) {
      if (screen) {
        this._stream = await navigator
            .getDisplayMedia(_buildMediaConstraints(false, true));
      } else {
        this._stream =
            await navigator.getUserMedia(_buildMediaConstraints(audio, video));
      }
    }
  }

  set mid(id) {
    this._mid = id;
  }

  String get mid => _mid;

  MediaStream get stream => _stream;

  _buildMediaConstraints(audio, video) {
    return {
      "audio": audio,
      "video": video
          ? {
              "mandatory": {
                "minWidth": '640',
                "minHeight": '480',
                "minFrameRate": '30',
              },
              "facingMode": "user",
              "optional": [],
            }
          : false
    };
  }
}
