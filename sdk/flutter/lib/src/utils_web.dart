
import 'package:flutter_webrtc/webrtc.dart';

addTransceiver(RTCPeerConnection pc, type, options){
    pc.addTransceiver(type, options);
}