import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'logger.dart' show Logger;
import 'room.dart';
import 'rtc.dart';

class SFU extends EventEmitter {
  var logger = new Logger("Pion::SFU");
  Room _room;
  RTC _rtc;

  SFU (url) {
        _room = new Room(url);
        _rtc = new RTC();
        // bind event callbaks.
        _room.on('onRoomConnect', _onRoomConnect);
        _room.on('onRoomDisconnect',_onRoomDisconnect);
        _room.on('onRtcCreateRecver', _onRtcCreateRecver);
        _room.on('onRtcLeaveRecver', _onRtcLeaveRecver);
    }

    close () {
        logger.debug('Force close.');
        _room.close();
    }

    join (roomId) {
        logger.debug('Join to [' + roomId + ']');
        _room.join(roomId);
    }

    publish () {
        _createSender(_room.uid);
    }

    leave () {
        _room.leave();
    }

    _onRoomConnect(){
        logger.debug('onRoomConnect');

        _rtc.on('localstream',(id, stream)  {
            this.emit('addLocalStream', id, stream);
        });

        _rtc.on('addstream',(id, stream)  {
            this.emit('addRemoteStream', id, stream);
        });

        _rtc.on('removestream',(id, stream)  {
            this.emit('removeRemoteStream', id, stream);
        });

        this.emit('connect');
    }

    _onRoomDisconnect(){
        logger.debug('onRoomDisconnect');
        this.emit('disconnect');
    }

    _createSender(pubid) async {
        try {
            var sender = await _rtc.createSender(pubid);
            sender.pc.onIceCandidate = (cand) async {
                if (!sender.senderOffer) {
                    var offer = sender.pc.getLocalDescription();
                    logger.debug('Send offer sdp => ' + offer.toString());
                    sender.senderOffer = true;
                    var answer = await _room.publish(offer,pubid);
                    var jsep = answer['jsep'];
                    var desc = RTCSessionDescription(jsep['sdp'], jsep['type']);
                    logger.debug('Got answer(' + pubid + ') sdp => ' + desc.sdp);
                    sender.pc.setRemoteDescription(desc);
                }
            };
            var desc = await sender.pc.createOffer({ 'offerToReceiveVideo': false, 'offerToReceiveAudio': false });
            sender.pc.setLocalDescription(desc);
        }catch(error){
            logger.debug('onCreateSender error => ' + error);
        }
    }

    _onRtcCreateRecver(pubid) async {
        try {
            var receiver = await _rtc.createRecver(pubid);
            receiver.pc.onIceCandidate = (cand) async {
                if (!receiver.senderOffer) {
                    var offer = receiver.pc.getLocalDescription();
                    logger.debug('Send offer sdp => ' + offer.toString());
                    receiver.senderOffer = true;
                    var answer = await _room.subscribe(offer,pubid);
                    var jsep = answer['jsep'];
                    var desc = RTCSessionDescription(jsep['sdp'], jsep['type']);
                    logger.debug('Got answer(' + pubid + ') sdp => ' + desc.sdp);
                    receiver.pc.setRemoteDescription(desc);
                }
            };
            var desc = await receiver.pc.createOffer({ 'offerToReceiveVideo': true, 'offerToReceiveAudio': true });
            receiver.pc.setLocalDescription(desc);
        }catch(error){
            logger.debug('onRtcCreateRecver error => ' + error);
        }
    }

    _onRtcLeaveRecver(pubid) {
        _rtc.closeRecver(pubid);
    }
}