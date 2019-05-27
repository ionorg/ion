import 'package:events2/events2.dart';
import 'package:protoo_client/protoo_client.dart';
import 'package:uuid/uuid.dart';
import 'logger.dart' show Logger;

const protooPort = 8443;

class Room extends EventEmitter {
    var logger = new Logger("Pion::Room");
    var _uuid = new Uuid();
    var _uid;
    var _rid;
    var _url;
    Peer _protoo;

    Room(url) {
        _uid = _uuid.v4();
        _url = url + '?peer=' + _uid;
        _protoo = new Peer(_url);

        _protoo.on('open', () {
            logger.debug('Peer [open] event');
            this.emit('onRoomConnect');
        });

        _protoo.on('disconnected', () {
            logger.debug('Peer [disconnected] event');
            this.emit('onRoomDisconnect');
        });

        _protoo.on('close', () {
            logger.debug('Peer [close] event');
            this.emit('onRoomDisconnect');
        });

        _protoo.on('request', _handleRequest);
        _protoo.on('notification', _handleNotification);
    }

    String get uid => _uid;

    join(roomId) async {
        _rid = roomId;
        try{
            var data = await _protoo.send('join', {'rid': _rid});
            logger.debug('join success: result => ' + data.toString());
        }catch(error) {
            logger.debug('join reject: error =>' + error);
        }
    }

    Future<dynamic> publish(offer, pubid) async {
        try {
            var answer = await _protoo.send('publish',{ 'jsep': { 'type': offer.type, 'sdp': offer.sdp}, 'pubid': pubid});
            logger.debug('publish success => ' + answer.toString());
            return answer;
        }catch(error) {
            throw error;
        }
    }

    Future<dynamic> subscribe(offer, pubid) async {
        try {
            var answer = await _protoo.send('subscribe',{'jsep': { 'type': offer.type, 'sdp': offer.sdp}, 'pubid': pubid});
            logger.debug('subscribe success => ' + answer.toString());
            return answer;
        }catch(error) {
            throw error;
        }
    }

    close() {
        _protoo.close();
    }

    leave() {
        _protoo.send('leave',{'rid': _rid});
    }

    _handleRequest(request, accept, reject) {
        logger.debug('Handle request from server: [method:' + request.method + ', data:' + request.data +']');
    }

    _handleNotification (notification) {
      logger.debug('Handle notification from server: [method:' + notification.method + ', data:' + notification.data +']');

        switch(notification.method){
            case 'onPublish':
                {
                    var pubid = notification.data.pubid;
                    logger.debug('Got publish from => ' + pubid);
                    this.emit('onRtcCreateRecver', pubid);
                    break;
                }
                case 'onUnpublish':
                {
                    var pubid = notification.data.pubid;
                    logger.debug('[' + pubid + ']  => leave !!!!');
                    this.emit('onRtcLeaveRecver', pubid);
                    break;
                }
        }
    }
}
