import 'dart:async';
import 'dart:convert';

import 'package:events2/events2.dart';
import 'package:flutter_webrtc/web/rtc_peerconnection_factory.dart';
import 'package:protoo_client/protoo_client.dart';
import 'package:sdp_transform/sdp_transform.dart' as sdpTransform;
import 'package:uuid/uuid.dart';

import 'logger.dart' show Logger;
import 'stream.dart';

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

    this._protoo.connect();
  }

  String get uid => _uid;

  Future<dynamic> join(roomId, info) async {
    this._rid = roomId;
    info = info ?? {'name': 'Guest'};
    try {
      var data = await this
          ._protoo
          .send('join', {'rid': this._rid, 'id': this._uid, 'info': info});
      logger.debug('join success: result => ' + _encoder.convert(data));
      return data;
    } catch (error) {
      logger.debug('join reject: error =>' + error);
    }
  }

  Future<dynamic> leave() async {
    try {
      var data =
          await this._protoo.send('leave', {'rid': this._rid, 'id': this._uid});
      logger.debug('leave success: result => ' + _encoder.convert(data));
      return data;
    } catch (error) {
      logger.debug('leave reject: error =>' + error);
    }
  }

  Future<Stream> publish([audio = true, video = true, screen = false, codec = 'vp8']) async {
    logger.debug('publish');
    Completer completer = new Completer();
    var pc;
    try {
      var stream = new Stream();
      await stream.init(true, audio, video, screen);
      pc = await this._createSender(stream.stream, codec);

      pc.onicecandidate = (e) async {
        if (!pc.sendOffer) {
          var offer = pc.localDescription;
          logger.debug('Send offer sdp => ' + offer.sdp);
          pc.sendOffer = true;
          var options = {
            'audio': audio,
            'video': video,
            'screen': screen,
            'codec': codec
          };
          var result = await this
              ._protoo
              .send('publish', {'jsep': offer, 'options': options});
          await pc.setRemoteDescription(result.jsep);
          logger.debug('publish success => ' + _encoder.convert(result));
          stream.mid = result.mid;
          this._pcs[stream.mid] = pc;
          completer.complete(stream);
        }
      };
    } catch (error) {
      logger.debug('publish request error  => ' + error);
      pc.close();
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

  Future<Stream> subscribe(rid, mid) async {
    logger.debug('subscribe rid => $rid, mid => $mid');
    Completer completer = new Completer();
    try {
      var pc = await this._createReceiver(mid);
      pc.onaddstream = (e) {
        var stream = e.stream;
        logger.debug('Stream::pc::onaddstream ' + stream.id);
        completer.complete(new Stream(mid, stream));
      };
      pc.onremovestream = (e) {
        var stream = e.stream;
        logger.debug('Stream::pc::onremovestream ' + stream.id);
      };
      pc.onicecandidate = (e) async {
        if (!pc.sendOffer) {
          var jsep = pc.localDescription;
          logger.debug('Send offer sdp => ' + jsep.sdp);
          pc.sendOffer = true;
          var result = await this
              ._protoo
              .send('subscribe', {'rid': rid, 'jsep': jsep, 'mid': mid});
          logger.debug('subscribe success => result($mid) sdp => ' +
              result.jsep.sdp);
          await pc.setRemoteDescription(result.jsep);
        }
      };
    } catch (error) {
      logger.debug('subscribe request error  => ' + error);
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
      logger.debug('unsubscribe reject: error =>' + error);
    }
  }

  close() {
    this._protoo.close();
  }

  _payloadModify(desc, codec) {
    if (codec == null) return desc;
    /*
         * DefaultPayloadTypePCMU = 0
         * DefaultPayloadTypePCMA = 8
         * DefaultPayloadTypeG722 = 9
         * DefaultPayloadTypeOpus = 111
         * DefaultPayloadTypeVP8  = 96
         * DefaultPayloadTypeVP9  = 98
         * DefaultPayloadTypeH264 = 102
        */
    var payload;
    var codeName = '';
    var session = sdpTransform.parse(desc.sdp);
    logger.debug('SDP object => $session');
    var videoIdx = -1;
    session['media'].map((m, index) {
      if (m.type == 'video') {
        videoIdx = index;
      }
    });

    if (videoIdx == -1) return desc;

    if (codec.toLowerCase() == 'vp8') {
      payload = DefaultPayloadTypeVP8;
      codeName = "VP8";
    } else if (codec.toLowerCase() == 'vp9') {
      payload = DefaultPayloadTypeVP9;
      codeName = "VP9";
    } else if (codec.toLowerCase() == 'h264') {
      payload = DefaultPayloadTypeH264;
      codeName = "H264";
    } else {
      return desc;
    }

    logger.debug('Setup codec => ' + codeName + ', payload => ' + payload);

    var rtp = [
      {"payload": payload, "codec": codeName, "rate": 90000, "encoding": null},
      {"payload": 97, "codec": "rtx", "rate": 90000, "encoding": null}
    ];

    session['media'][videoIdx]["payloads"] = payload + " 97";
    session['media'][videoIdx]["rtp"] = rtp;

    var fmtp = [
      {"payload": 97, "config": "apt=" + payload}
    ];

    session['media'][videoIdx]["fmtp"] = fmtp;

    var rtcpFB = [
      {"payload": payload, "type": "transport-cc", "subtype": null},
      {"payload": payload, "type": "ccm", "subtype": "fir"},
      {"payload": payload, "type": "nack", "subtype": null},
      {"payload": payload, "type": "nack", "subtype": "pli"}
    ];
    session['media'][videoIdx]["rtcpFb"] = rtcpFB;

    var tmp = desc;
    tmp.sdp = sdpTransform.write(session, null);
    return tmp;
  }

  _createSender(stream, codec) async {
    logger.debug('create sender => $codec');
    var pc = await createPeerConnection(_iceServers, _config);
    //pc.sendOffer = false;
    pc.addStream(stream);
    var offer = await pc.createOffer({
      'mandatory': {
        'OfferToReceiveAudio': false,
        'OfferToReceiveVideo': false,
      },
      'optional': [],
    });
    var desc = this._payloadModify(offer, codec);
    pc.setLocalDescription(desc);
    return pc;
  }

  _createReceiver(uid) async {
    logger.debug('create receiver => $uid');
    var pc = await createPeerConnection(_iceServers, _config);
    //pc.sendOffer = false;
    //pc.addTransceiver('audio', { 'direction': 'recvonly' });
    //pc.addTransceiver('video', { 'direction': 'recvonly' });
    var desc = await pc.createOffer({
      'mandatory': {
        'OfferToReceiveAudio': true,
        'OfferToReceiveVideo': true,
      },
      'optional': [],
    });
    pc.setLocalDescription(desc);
    this._pcs[uid] = pc;
    return pc;
  }

  _removePC(id) {
    var pc = this._pcs[id];
    if (pc) {
      logger.debug('remove pc mid => $id');
      pc.close();
      this._pcs.remove(id);
    }
  }

  _handleRequest(request, accept, reject) {
    logger.debug(
        'Handle request from server: [method:${request['method']}, data:${request['data']}]');
  }

  _handleNotification(notification) {
    var method = notification['method'];
    var data = notification['data'];
    logger.debug(
        'Handle notification from server: [method:$method, data:$data]');
    switch (method) {
      case 'peer-join':
        {
          var rid = data['rid'];
          var id = data['id'];
          var info = data['info'];
          logger.debug('peer-join peer rid => $rid, id => $id, info => $info');
          this.emit('peer-join', rid, id, info);
          break;
        }
      case 'peer-leave':
        {
          var rid = data['rid'];
          var id = data['id'];
          logger.debug('peer-leave peer rid => $rid, id => $id');
          this.emit('peer-leave', rid, id);
          break;
        }
      case 'stream-add':
        {
          var rid = data['rid'];
          var mid = data['mid'];
          var info = data['info'];
          logger
              .debug('stream-add peer rid => $mid, id => $mid, info => $info');
          this.emit('stream-add', rid, mid, info);
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
    }
  }
}
