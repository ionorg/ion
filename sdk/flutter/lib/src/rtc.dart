import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'logger.dart' show Logger;

class Sender {
  Sender(this.pc, this.stream, this.id);
  String id;
  RTCPeerConnection pc;
  bool senderOffer = false;
  MediaStream stream;
}

class Receiver {
  Receiver(this.publid);
  RTCPeerConnection pc;
  bool senderOffer = false;
  String publid;
  List<MediaStream> streams = new List();
}

final Map<String, dynamic> configuration = {
    "iceServers": [
      {"url": "stun:stun.stunprotocol.org:3478"},
    ],
    'sdpSemantics': 'plan-b',
  };

final Map<String, dynamic> mediaConstraints = {
      "audio": true,
      "video": {
        "mandatory": {
          "minWidth": '640',
          "minHeight": '480',
          "minFrameRate": '30',
        },
        "facingMode": "user",
        "optional": [],
      }
    };

class RTC extends EventEmitter {
  var logger = new Logger("Pion::RTC");
  Sender _sender;
  Map<String, Receiver> _receivers = new Map();

  RTC();

  get sender => _sender;

  get receivers => _receivers;

  Future<Sender> createSender(var pubid) async {
        final Map<String, dynamic> constraints = {
          'mandatory': {
            'OfferToReceiveAudio': false,
            'OfferToReceiveVideo': false,
          },
          'optional': [],
        };
        var pc = await createPeerConnection(configuration, constraints);
        var stream = await navigator.getUserMedia(mediaConstraints);
        pc.addStream(stream);
        this.emit('localstream', pubid, stream);
        _sender = Sender(pc, stream,pubid);
        return _sender;
    }

  Future<Receiver> createRecver(pubid) async {
      final Map<String, dynamic> _constraints = {
          'mandatory': {
            'OfferToReceiveAudio': true,
            'OfferToReceiveVideo': true,
          },
          'optional': [],
        };
        try {
            var receiver = Receiver(pubid);
            RTCPeerConnection pc = await createPeerConnection(configuration, _constraints);
            pc.onAddStream = (stream){
                logger.debug('receiver.pc.onaddstream = ' + stream.id);
                var receiver = _receivers[pubid];
                receiver.streams.add(stream);
                this.emit('addstream', pubid, stream);
            };
            pc.onRemoveStream = (stream) {
                logger.debug('receiver.pc.onremovestream = ' + stream.id);
                _receivers.remove(pubid);
                this.emit('removestream', pubid, stream);
            };
            receiver.pc = pc;
            _receivers[pubid] = receiver;
            return receiver;
        } catch (e) {
            logger.debug(e);
            throw e;
        }
    }

    closeSender() async {
      if(_sender != null){
        this.emit('removelocalstream', _sender.id, _sender.stream);
        await _sender.stream.dispose();
        await _sender.pc.close();
        _sender = null;
      }
    }

    closeReceiver(pubid) {
        var receiver = _receivers[pubid];
        if(receiver != null) {
            receiver.streams.forEach((stream) {
                this.emit('removestream', pubid, stream);
            });
            receiver.pc.close();
            _receivers.remove(pubid);
        }
    }

    closeReceivers() async {
      _receivers.forEach((pubid, receiver) async {
            receiver.streams.forEach((stream) async {
                await stream.dispose();
                this.emit('removestream', pubid, stream);
            });
            await receiver.pc.close();
      });
    }
}
