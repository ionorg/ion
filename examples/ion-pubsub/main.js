const localVideo = document.getElementById("local-video");
const remotesDiv = document.getElementById("remotes");

/* eslint-env browser */
const joinBtn = document.getElementById("join-btn");
const leaveBtn = document.getElementById("leave-btn");
const publishBtn = document.getElementById("publish-btn");
const publishSBtn = document.getElementById("publish-simulcast-btn");

const codecBox = document.getElementById("select-box1");
const resolutionBox = document.getElementById("select-box2");
const simulcastBox = document.getElementById("check-box");
const localData = document.getElementById("local-data");
const remoteData = document.getElementById("remote-data");
const remoteSignal= document.getElementById("remote-signal");
const subscribeBox = document.getElementById("select-box3");
const sizeTag = document.getElementById("size-tag");
const brTag = document.getElementById("br-tag");
let localDataChannel;
let trackEvent;

const url = 'http://localhost:5551';
const uid = uuidv4();
const sid = "ion";
let room;
let rtc;
let localStream;
let start;

const join = async () => {
    console.log("[join]: sid="+sid+" uid=", uid)
    const connector = new Ion.Connector(url, "token");
    
    connector.onopen = function (service){
        console.log("[onopen]: service = ", service.name);
    };

    connector.onclose = function (service){
        console.log('[onclose]: service = ' + service.name);
    };


    room = new Ion.Room(connector);
    
    room.onjoin = function (result){
        console.log('[onjoin]: success ' + result.success + ', room info: ' + JSON.stringify(result.room));
    };
    
    room.onleave = function (reason){
        console.log('[onleave]: leave room, reason ' + reason);
    };
    
    room.onmessage = function (msg){
        console.log('[onmessage]: Received msg:',  msg)
        const uint8Arr = new Uint8Array(msg.data);
        const decodedString = String.fromCharCode.apply(null, uint8Arr);
        const json  = JSON.parse(decodedString);
        remoteData.innerHTML = remoteData.innerHTML + json.msg+ '\n';
    };
    
    room.onpeerevent = function (event){
        switch(event.state) {
            case Ion.PeerState.JOIN:
                console.log('[onpeerevent]: Peer ' + event.peer.uid + ' joined');
                break;
            case Ion.PeerState.LEAVE:
                console.log('[onpeerevent]: Peer ' + event.peer.uid + ' left');
                break;
            case Ion.PeerState.UPDATE:
                console.log('[onpeerevent]: Peer ' + event.peer.uid + ' updated');
                break;
        }
    };
    
    room.onroominfo = function (info){
        console.log('[onroominfo]: ' + JSON.stringify(info));
    };
    
    room.ondisconnect = function (dis){
        console.log('[ondisconnect]: Disconnected from server ' + dis);
    };
    
    const result = await room.join({
        sid: sid,
        uid: uid,
        displayname: 'new peer',
        extrainfo: '',
        destination: 'webrtc://ion/peer1',
        role: Ion.Role.HOST,
        protocol: Ion.Protocol.WEBRTC ,
        avatar: 'string',
        direction: Ion.Direction.INCOMING,
        vendor: 'string',
    }, '')
        .then((result) => {
            console.log('[join] result: success ' + result?.success + ', room info: ' + JSON.stringify(result?.room));
            joinBtn.disabled = "true";
            remoteData.innerHTML = remoteData.innerHTML + JSON.stringify(result) + '\n';
            leaveBtn.removeAttribute('disabled');
            publishBtn.removeAttribute('disabled');
            publishSBtn.removeAttribute('disabled');

            rtc = new Ion.RTC(connector);

            rtc.ontrack = (track, stream) => {
              console.log("got ", track.kind, " track", track.id, "for stream", stream.id);
              if (track.kind === "video") {
                track.onunmute = () => {
                  if (!streams[stream.id]) {
                    const remoteVideo = document.createElement("video");
                    remoteVideo.srcObject = stream;
                    remoteVideo.autoplay = true;
                    remoteVideo.muted = true;
                    remoteVideo.addEventListener("loadedmetadata", function () {
                      sizeTag.innerHTML = `${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`;
                    });
            
                    remoteVideo.onresize = function () {
                      sizeTag.innerHTML = `${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`;
                    };
                    remotesDiv.appendChild(remoteVideo);
                    streams[stream.id] = stream;
                    stream.onremovetrack = () => {
                      if (streams[stream.id]) {
                        remotesDiv.removeChild(remoteVideo);
                        streams[stream.id] = null;
                      }
                    };
                    getStats();
                  }
                };
              }
            };

            rtc.ontrackevent = function (ev) {
              console.log("ontrackevent: \nuid = ", ev.uid, " \nstate = ", ev.state, ", \ntracks = ", JSON.stringify(ev.tracks));
              if (trackEvent === undefined) {
                console.log("store trackEvent=", ev)
                trackEvent = ev;
              }
              remoteSignal.innerHTML = remoteSignal.innerHTML + JSON.stringify(ev) + '\n';
            };

            rtc.join(sid, uid);

            const streams = {};

            start = (sc) => {
              publishSBtn.disabled = "true";
              publishBtn.disabled = "true";

              let constraints = {
                resolution: resolutionBox.options[resolutionBox.selectedIndex].value,
                codec: codecBox.options[codecBox.selectedIndex].value,
                audio: true,
                simulcast: sc,
              }
              console.log("getUserMedia constraints=", constraints)
              Ion.LocalStream.getUserMedia(constraints)
                .then((media) => {
                  localStream = media;
                  localVideo.srcObject = media;
                  localVideo.autoplay = true;
                  localVideo.controls = true;
                  localVideo.muted = true;

                  rtc.publish(media);
                  localDataChannel = rtc.createDataChannel(uid);
                })
                .catch(console.error);
            };
            
    });
}

const send = () => {
    if (!room) {
        alert('join room first!', '', {
        confirmButtonText: 'OK',
      });
      return
    }
    const payload = new Map();
    payload.set('msg', localData.value);
    console.log("[send]: sid=", sid, "from=", 'sender', "to=", uid, "payload=", payload);
    room.message(sid, uid, "all", 'Map', payload);
}


const leave = () => {
    console.log("[leave]: sid=" + sid + " uid=", uid)
    room.leave(sid, uid);
    joinBtn.removeAttribute('disabled');
    leaveBtn.disabled = "true";
    publishBtn.disabled = "true";
    publishSBtn.disabled = "true";
    location.reload();
}

const subscribe = () => {
    let layer = subscribeBox.value
    console.log("subscribe trackEvent=", trackEvent, "layer=", layer)
    var infos = [];
    trackEvent.tracks.forEach(t => {
        if (t.layer === layer && t.kind === "video"){
          infos.push({
            track_id: t.id,
            mute: t.muted,
            layer: t.layer,
            subscribe: true
          });
        }
        
        if (t.kind === "audio"){
          infos.push({
            track_id: t.id,
            mute: t.muted,
            layer: t.layer,
            subscribe: true
          });
        }
    });
    console.log("subscribe infos=", infos)
    rtc.subscribe(infos);
}



const controlLocalVideo = (radio) => {
  if (radio.value === "false") {
    localStream.mute("video");
  } else {
    localStream.unmute("video");
  }
};

const controlLocalAudio = (radio) => {
  if (radio.value === "false") {
    localStream.mute("audio");
  } else {
    localStream.unmute("audio");
  }
};

const getStats = () => {
  let bytesPrev;
  let timestampPrev;
  setInterval(() => {
    rtc.getSubStats(null).then((results) => {
      results.forEach((report) => {
        const now = report.timestamp;

        let bitrate;
        if (
          report.type === "inbound-rtp" &&
          report.mediaType === "video"
        ) {
          const bytes = report.bytesReceived;
          if (timestampPrev) {
            bitrate = (8 * (bytes - bytesPrev)) / (now - timestampPrev);
            bitrate = Math.floor(bitrate);
          }
          bytesPrev = bytes;
          timestampPrev = now;
        }
        if (bitrate) {
          brTag.innerHTML = `${bitrate} kbps @ ${report.framesPerSecond} fps`;
        }
      });
    });
  }, 1000);
};

function syntaxHighlight(json) {
  json = json
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
  return json.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
    function (match) {
      let cls = "number";
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = "key";
        } else {
          cls = "string";
        }
      } else if (/true|false/.test(match)) {
        cls = "boolean";
      } else if (/null/.test(match)) {
        cls = "null";
      }
      return '<span class="' + cls + '">' + match + "</span>";
    }
  );
}
