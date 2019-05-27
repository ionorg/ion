import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'logger.dart' show Logger;

const ices = 'stun:stun.stunprotocol.org:3478';

class Sender {
  Sender(){}
  RTCPeerConnection pc;
  bool senderOffer = false;
}

class Receiver {
  Receiver(this.publid);
  RTCPeerConnection pc;
  bool senderOffer = false;
  String publid;
  var streams = [];
}

class RTC extends EventEmitter {
  var logger = new Logger("Pion::RTC");
  Sender _sender;
  Map<String, Receiver> _receivers = new Map();

  RTC();

  get sender => _sender;

  get receivers => _receivers;

  Future<Sender> createSender(var pubid) async {
        var sender  = Sender();
        _sender.pc = await createPeerConnection({ 'iceServers': [{ 'urls': ices }]},{});
        var stream = await navigator.getUserMedia({ 'video': true, 'audio': true });
        _sender.pc.addStream(stream);
        this.emit('localstream', pubid, stream);
        return sender;
    }

  Future<Receiver> createRecver(pubid) async {
        try {
            var receiver = Receiver(pubid);
            RTCPeerConnection pc = await createPeerConnection({ 'iceServers': [{ 'urls': ices }]},{});
            pc.onIceCandidate = (candidate) {
                logger.debug('receiver.pc.onicecandidate => ' + candidate.toString());
            };

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

    closeRecver(pubid) {
        var receiver = _receivers[pubid];
        if(receiver != null) {
            receiver.streams.forEach((stream) {
                this.emit('removestream', pubid, stream);
            });
            receiver.pc.close();
            _receivers.remove(pubid);
        }
    }
}
