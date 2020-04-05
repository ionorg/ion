import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:events2/events2.dart';
import 'package:flutter_webrtc/webrtc.dart';
import 'package:protoo_client/protoo_client.dart';
import 'package:sdp_transform/sdp_transform.dart' as sdpTransform;
import 'package:uuid/uuid.dart';

import 'logger.dart' show Logger;
import 'stream.dart';
import 'utils.dart' if (dart.library.js) 'utils_web.dart';

const DefaultPayloadTypePCMU = 0;
const DefaultPayloadTypePCMA = 8;
const DefaultPayloadTypeG722 = 9;
const DefaultPayloadTypeOpus = 111;
const DefaultPayloadTypeVP8 = 96;
const DefaultPayloadTypeVP9 = 98;
const DefaultPayloadTypeH264 = 102;

class Client extends EventEmitter {
  JsonEncoder _encoder = JsonEncoder();
  var logger = new Logger("Ion::Client");
  var _uuid = new Uuid();
  var _pcs = new Map();
  var _uid;
  var _rid;
  var _url;
  Peer _protoo;

  Map<String, dynamic> _iceServers = {
    'iceServers': [
      {'url': 'stun:stun.stunprotocol.org:3478'},
      /*
       * turn server configuration example.
      {
        'url': 'turn:123.45.67.89:3478',
        'username': 'change_to_real_user',
        'credential': 'change_to_real_secret'
      },
       */
    ]
  };

  final Map<String, dynamic> _config = {
    'mandatory': {},
    'optional': [
      {'DtlsSrtpKeyAgreement': true},
    ],
  };

  Client(url) {
    _uid = _uuid.v4();
    _url = url + '?peer=' + _uid;
    _protoo = new Peer(_url);

    this._protoo.on('open', () {
      logger.debug('Peer "open" event');
      this.emit('transport-open');
    });

    this._protoo.on('disconnected', () {
      logger.debug('Peer "disconnected" event');
      this.emit('transport-failed');
    });

    this._protoo.on('close', () {
      logger.debug('Peer "close" event');
      this.emit('transport-closed');
    });

    this._protoo.on('request', this._handleRequest);
    this._protoo.on('notification', this._handleNotification);
  }

  connect() async => this._protoo.connect();

  String get uid => _uid;

  Future<dynamic> join(roomId, info) async {
    this._rid = roomId;
    info = info ?? {'name': 'Guest'};
    try {
      var data = await this
          ._protoo
          .send('join', {'rid': this._rid, 'uid': this._uid, 'info': info});
      logger.debug('join success: result => ' + _encoder.convert(data));
      return data;
    } catch (error) {
      logger.debug('join reject: error =>' + error);
    }
  }

  Future<dynamic> leave() async {
    try {
      var data = await this
          ._protoo
          .send('leave', {'rid': this._rid, 'uid': this._uid});
      logger.debug('leave success: result => ' + _encoder.convert(data));
      return data;
    } catch (error) {
      logger.debug('leave reject: error =>' + error);
    }
  }

  Future<Stream> publish(
      [audio = true,
      video = true,
      screen = false,
      codec = 'vp8',
      bandwidth = 512,
      quality = 'hd']) async {
    logger.debug('publish');
    Completer completer = new Completer<Stream>();
    RTCPeerConnection pc;
    try {
      var stream = new Stream();
      await stream.init(true, audio, video, screen, quality);
      logger.debug('create sender => $codec');
      pc = await createPeerConnection(_iceServers, _config);
      await pc.addStream(stream.stream);
      bool sendOffer = false;
      pc.onIceCandidate = (candidate) async {
        if (sendOffer == false) {
          sendOffer = true;
          var offer = await pc.getLocalDescription();
          logger.debug('Send offer sdp => ' + offer.sdp);
          var options = {
            'audio': audio,
            'video': video,
            'screen': screen,
            'codec': codec,
            'bandwidth': bandwidth,
          };
          var result = await this._protoo.send('publish',
              {'rid': this._rid, 'jsep': offer.toMap(), 'options': options});
          await pc.setRemoteDescription(RTCSessionDescription(
              result['jsep']['sdp'], result['jsep']['type']));
          logger.debug('publish success => ' + _encoder.convert(result));
          stream.mid = result['mid'];
          this._pcs[stream.mid] = pc;
          completer.complete(stream);
        }
      };
      var offer = await pc.createOffer({
        'mandatory': {
          'OfferToReceiveAudio': false,
          'OfferToReceiveVideo': false,
        },
        'optional': [],
      });
      var desc = this._payloadModify(offer, codec, true);
      await pc.setLocalDescription(desc);
    } catch (error) {
      logger.debug('publish request error  => ' + error);
      if (pc != null) {
        pc.close();
      }
      completer.completeError(error);
    }
    return completer.future;
  }

  Future<dynamic> unpublish(mid) async {
    logger.debug('unpublish rid => ${this._rid}, mid => $mid');
    this._removePC(mid);
    try {
      var data =
          await this._protoo.send('unpublish', {'rid': this._rid, 'mid': mid});
      logger.debug('unpublish success: result => ' + _encoder.convert(data));
      return data;
    } catch (error) {
      logger.debug('unpublish reject: error =>' + error);
    }
  }

  Future<Stream> subscribe(rid, mid,
      tracks, [String bandwidth = '512']) async {
    logger.debug('subscribe rid => $rid, mid => $mid,  tracks => ${tracks.toString()}');
    Completer completer = new Completer<Stream>();
    var codec = "";
    tracks?.forEach((trackID,trackInfoArr) async {
        logger.debug('trackInfoArr=$trackInfoArr');

        for(var i=0; i<trackInfoArr.length; i++){
          var trackInfo=trackInfoArr[i];
          logger.debug('trackInfo=$trackInfo');
          var type = trackInfo['type'];
          logger.debug('type=$type');
          if (type == "video") {
            codec = trackInfo['codec'];
            logger.debug('codec=$codec');
          }
        }
      }
    );

     
    var options = {
      'codec': codec,
      'bandwidth': bandwidth,
    };
    try {
      logger.debug('create receiver => $mid');
      RTCPeerConnection pc = await createPeerConnection(_iceServers, _config);
      bool sendOffer = false;
      var sub_mid = "";
      pc.onAddStream = (stream) {
        logger.debug('Stream::pc::onaddstream ' + stream.id);
        completer.complete(new Stream(sub_mid, stream));
      };
      pc.onRemoveStream = (stream) {
        logger.debug('Stream::pc::onremovestream ' + stream.id);
      };
      pc.onIceCandidate = (candidate) async {
        if (sendOffer == false) {
          sendOffer = true;
          RTCSessionDescription jsep = await pc.getLocalDescription();
          logger.debug('Send offer sdp => ' + jsep.sdp);
          var result = await this._protoo.send('subscribe', {
            'rid': rid,
            'jsep': jsep.toMap(),
            'mid': mid,
            'options': options
          });
          sub_mid = result['mid'];
          logger.debug('subscribe success => result(mid => $sub_mid) sdp => ' +
              result['jsep']['sdp']);
          await pc.setRemoteDescription(RTCSessionDescription(
              result['jsep']['sdp'], result['jsep']['type']));
        }
      };

      addTransceiver(pc, "audio", {"direction": 'recvonly'});
      addTransceiver(pc, "video", {"direction": 'recvonly'});

      var offer = await pc.createOffer({
        'mandatory': {
          'OfferToReceiveAudio': true,
          'OfferToReceiveVideo': true,
        },
        'optional': [],
      });
      var desc = this._payloadModify(offer, codec, false);
      await pc.setLocalDescription(desc);
      this._pcs[mid] = pc;
    } catch (error) {
      logger.debug('subscribe request error  => ' + error.toString());
      completer.completeError(error);
    }

    return completer.future;
  }

  Future<dynamic> unsubscribe(rid, mid) async {
    logger.debug('unsubscribe rid => $rid, mid => $mid');
    try {
      var data =
          await this._protoo.send('unsubscribe', {'rid': rid, 'mid': mid});
      logger.debug('unsubscribe success: result => ' + _encoder.convert(data));
      this._removePC(mid);
      return data;
    } catch (error) {
      logger.debug('unsubscribe reject: error =>' + error.toString());
      this._removePC(mid);
    }
  }

  Future<dynamic> broadcast(rid, info) async {
    try {
      var data = await this
          ._protoo
          .send('broadcast', {'rid': this._rid, 'uid': this._uid, 'info': info});
      logger.debug('broadcast success: result => ' + _encoder.convert(data));
      return data;
    } catch (error) {
      logger.debug('broadcast reject: error =>' + error);
    }
  }

  close() {
    this._protoo.close();
  }

  _payloadModify(desc, codec, sender) {
    if (codec == null) return desc;

    logger.debug('SDP string => ${desc.sdp}');
    var session = sdpTransform.parse(desc.sdp);
    //logger.debug('SDP object => $session');

    var audioIndex = session['media'].indexWhere((e) => e['type'] == 'audio');
    if (audioIndex != -1) {
      var codeName = "OPUS";
      var payload = 111;
      logger.debug('Setup audio codec => $codeName, payload => $payload');
      var rtp = [
        {"payload": payload, "codec": codeName, "rate": 48000, "encoding": 2},
      ];
      var fmtp = [
        {"payload": payload, "config": "minptime=10;useinbandfec=1"}
      ];

      session['media'][audioIndex]["payloads"] = '$payload';
      session['media'][audioIndex]["rtp"] = rtp;
      session['media'][audioIndex]["fmtp"] = fmtp;

      if (sender) {
        session['media'][audioIndex]["direction"] = "sendonly";
      } else {
        session['media'][audioIndex]["direction"] = "recvonly";
      }
    }

    var videoIdx = session['media'].indexWhere((e) => e['type'] == 'video');

    if (videoIdx != -1) {
      var payload;
      var rtx = 97;
      var codeName = '';
      if (codec.toLowerCase() == 'vp8') {
        payload = DefaultPayloadTypeVP8;
        codeName = "VP8";
      } else if (codec.toLowerCase() == 'vp9') {
        payload = DefaultPayloadTypeVP9;
        codeName = "VP9";
      } else if (codec.toLowerCase() == 'h264') {
        payload = 102;
        codeName = "H264";
      } else {
        return desc;
      }

      logger.debug('Setup video codec => $codeName, payload => $payload');

      var rtp = [
        {
          "payload": payload,
          "codec": codeName,
          "rate": 90000,
          "encoding": null
        },
        //{"payload": rtx, "codec": "rtx", "rate": 90000, "encoding": null}
      ];

      var fmtp = [
        //{"payload": rtx, "config": "apt=$payload"}
      ];

      if (payload == DefaultPayloadTypeH264) {
        fmtp.add({
          "payload": payload,
          "config":
              "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"
        });
      }

      var rtcpFB = [
        {"payload": payload, "type": "goog-remb", "subtype": null},
        {"payload": payload, "type": "transport-cc", "subtype": null},
        {"payload": payload, "type": "ccm", "subtype": null},
        {"payload": payload, "type": "ccm", "subtype": "fir"},
        {"payload": payload, "type": "nack", "subtype": null},
        {"payload": payload, "type": "nack", "subtype": "pli"}
      ];

      session['media'][videoIdx]["payloads"] = '$payload'; // $rtx';
      session['media'][videoIdx]["rtp"] = rtp;
      session['media'][videoIdx]["fmtp"] = fmtp;
      session['media'][videoIdx]["rtcpFb"] = rtcpFB;

      if (session['media'][videoIdx]['ssrcGroups'] != null) {
        var ssrcGroup = session['media'][videoIdx]['ssrcGroups'][0];
        var ssrcs = ssrcGroup['ssrcs'];
        var videoSsrc = ssrcs.split(" ")[0];
        logger.debug('ssrcs => $ssrcs, video $videoSsrc');

        List newSsrcs = session['media'][videoIdx]['ssrcs'] as List;
        newSsrcs.removeWhere((item) => '${item['id']}' != videoSsrc);

        session['media'][videoIdx]['ssrcGroups'] = [];
        session['media'][videoIdx]['ssrcs'] = newSsrcs;
      }

      if (sender) {
        session['media'][videoIdx]["direction"] = "sendonly";
      } else {
        session['media'][videoIdx]["direction"] = "recvonly";
      }
    }

    /*else {
      List<Map<String, dynamic>> payloadMap = [
        {
          'codec': "VP8",
          'payload': DefaultPayloadTypeVP8,
          'rtx': 97,
        },
        {
          'codec': "VP9",
          'payload': DefaultPayloadTypeVP9,
          'rtx': 124,
        },
        {
          'codec': "H264",
          'payload': DefaultPayloadTypeH264,
          'rtx': 125,
        }
      ];

      var payloads = "";
      var rtps = [];
      var fmtps = [];
      var rtcpFBs = [];

      payloadMap.map((e) {
        var name = e['name'];
        var payload = e['payload'];
        var rtx = e['rtx'];

        payloads += '$payload $rtx';

        rtps.add({
          "payload": payload,
          "codec": name,
          "rate": 90000,
          "encoding": null
        });
        rtps.add(
            {"payload": rtx, "codec": "rtx", "rate": 90000, "encoding": null});

        fmtps.add({"payload": rtx, "config": "apt=$payload"});

        rtcpFBs.addAll([
          {"payload": payload, "type": "transport-cc", "subtype": null},
          {"payload": payload, "type": "ccm", "subtype": "fir"},
          {"payload": payload, "type": "nack", "subtype": null},
          {"payload": payload, "type": "nack", "subtype": "pli"}
        ]);
      });

      session['media'][videoIdx]["payloads"] = payloads;
      session['media'][videoIdx]["rtp"] = rtps;
      session['media'][videoIdx]["fmtp"] = fmtps;
      session['media'][videoIdx]["rtcpFb"] = rtcpFBs;
    }*/

    var tmp = desc;
    tmp.sdp = sdpTransform.write(session, null);
    logger.debug('SDP => ${tmp.sdp}');
    return tmp;
  }

  _removePC(mid) {
    RTCPeerConnection pc = this._pcs[mid];
    if (pc != null) {
      logger.debug('remove pc mid => $mid');
      pc.dispose();
      pc.close();
      this._pcs.remove(mid);
    }
  }

  _handleRequest(request, accept, reject) {
    logger.debug(
        'Handle request from server: [method:${request['method']}, data:${request['data']}]');
  }

  _handleNotification(notification) {
    var method = notification['method'];
    var data = notification['data'];
    logger
        .debug('Handle notification from server: [method:$method, data:$data]');
    switch (method) {
      case 'peer-join':
        {
          var rid = data['rid'];
          var uid = data['uid'];
          var info = data['info'];
          logger.debug(
              'peer-join peer rid => $rid, uid => $uid, info => ${info.toString()}');
          this.emit('peer-join', rid, uid, info);
          break;
        }
      case 'peer-leave':
        {
          var rid = data['rid'];
          var uid = data['uid'];
          logger.debug('peer-leave peer rid => $rid, uid => $uid');
          this.emit('peer-leave', rid, uid);
          break;
        }
      case 'stream-add':
        {
          var rid = data['rid'];
          var mid = data['mid'];
          var info = data['info'];
          var tracks = data['tracks'];
          logger.debug(
              'stream-add peer rid => $rid, mid => $mid, info => ${info.toString()},  tracks => $tracks');
          this.emit('stream-add', rid, mid, info, tracks);
          break;
        }
      case 'stream-remove':
        {
          var rid = data['rid'];
          var mid = data['mid'];
          logger.debug('stream-remove peer rid => $rid, mid => $mid');
          this.emit('stream-remove', rid, mid);
          this._removePC(mid);
          break;
        }
     case 'broadcast':
        {
          var rid = data['rid'];
          var uid = data['uid'];
          var info = data['info'];
          logger.debug(
              'broadcast peer rid => $rid, uid => $uid, info => ${info.toString()}');
          this.emit('broadcast', rid, uid, info);
          break;
        }
    }
  }
}
