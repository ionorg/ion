import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'package:sdp_transform/sdp_transform.dart' as sdp_transform;
import 'dart:convert';

import 'logger.dart' show Logger;
import 'room.dart';
import 'rtc.dart';

class SFU extends EventEmitter {
  JsonEncoder _jsonEnc = new JsonEncoder();
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

    /*Replace the payload to adapt SFU-WS */
  replacePayload(description) {
    return description;
    var session = sdp_transform.parse(description.sdp);
    print('session => ' + _jsonEnc.convert(session));

    var videoIdx = 1;

    /*
     * DefaultPayloadTypeG722 = 9
     * DefaultPayloadTypeOpus = 111
     * DefaultPayloadTypeVP8  = 96
     * DefaultPayloadTypeVP9  = 98
     * DefaultPayloadTypeH264 = 100
    */
    /*Add VP8 and RTX only.*/
    var rtp = [
      {"payload": 96, "codec": "VP8", "rate": 90000, "encoding": null},
      {"payload": 97, "codec": "rtx", "rate": 90000, "encoding": null}
    ];

    session['media'][videoIdx]["payloads"] = "96 97";
    session['media'][videoIdx]["rtp"] = rtp;

    var fmtp = [
      {"payload": 97, "config": "apt=96"}
    ];

    session['media'][videoIdx]["fmtp"] = fmtp;

    var rtcpFB = [
      {"payload": 96, "type": "transport-cc", "subtype": null},
      {"payload": 96, "type": "ccm", "subtype": "fir"},
      {"payload": 96, "type": "nack", "subtype": null},
      {"payload": 96, "type": "nack", "subtype": "pli"}
    ];
    session['media'][videoIdx]["rtcpFb"] = rtcpFB;

    var sdp = sdp_transform.write(session, null);
    return new RTCSessionDescription(sdp, description.type);
  }

    Future<dynamic> join(roomId) async {
        logger.debug('Join to [' + roomId + ']');
        return _room.join(roomId);
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

                    sender.senderOffer = true;
                }
            };
            sender.pc.onIceGatheringState = (state) async {
              if(state == RTCIceGatheringState.RTCIceGatheringStateComplete){
                 var offer = await sender.pc.getLocalDescription();
                    //logger.debug('Send offer sdp => ' + offer.toString());
                var answer = await _room.publish(offer,pubid);
                var jsep = answer['jsep'];
                var desc = RTCSessionDescription(jsep['sdp'], jsep['type']);
                logger.debug('Got answer(' + pubid + ') sdp => ' + desc.sdp);
                sender.pc.setRemoteDescription(desc);
              }
            };
            var tmp = await sender.pc.createOffer({ 'offerToReceiveVideo': false, 'offerToReceiveAudio': false });
            var offer = replacePayload(tmp);
            sender.pc.setLocalDescription(offer);
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
            var tmp = await receiver.pc.createOffer({ 'offerToReceiveVideo': true, 'offerToReceiveAudio': true });
            var offer = replacePayload(tmp);
            receiver.pc.setLocalDescription(offer);
        }catch(error){
            logger.debug('onRtcCreateRecver error => ' + error);
        }
    }

    _onRtcLeaveRecver(pubid) {
        _rtc.closeRecver(pubid);
    }
}