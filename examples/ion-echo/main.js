const connectBtn = document.getElementById("connect-btn");
const joinBtn = document.getElementById("join-btn");
const leaveBtn = document.getElementById("leave-btn");
const publishBtn = document.getElementById("publish-btn");
const publishSBtn = document.getElementById("publish-simulcast-btn");

const localVideo = document.getElementById("local-video");
const remoteVideo = document.getElementById("remote-video");
const localData = document.getElementById("local-data");
const remoteData = document.getElementById("remote-data");
const sizeTag = document.getElementById("size-tag");
const brTag = document.getElementById("br-tag");
const codecBox = document.getElementById("select-box1");
const resolutionBox = document.getElementById("select-box2");

const turnUrl = document.getElementById("turnUrl");
const turnUsername = document.getElementById("turnUsername");
const turnCredential = document.getElementById("turnCredential");
const relayBox = document.getElementById("select-box-relay");

const rttTag = document.getElementById("rtt-tag");

let simulcast = false;
let localDataChannel;

let url = window.location.href.replace(/\/+$/, '') + ':5551';
let sid = 'ion';
let uid = "local-user";
let connector;
let room;

let localRTC
let remoteStream

const connect = async () => {
  console.log("[connect]: turn server=", turnUrl.value);

  console.log("TURN server: "+turnUrl.value);

  const local = new Ion.Connector(url);
  const remote = new Ion.Connector(url);

  local.onopen = function (service) {
    console.log("[onopen]: service = ", service.name);
  };

  local.onclose = function (service, err) {
    console.log("[onclose]: service = ", service.name, ", err = ", JSON.stringify(err.detail));
  };

  let config = {
    codec: codecBox.value
  };

  if (turnUsername.value) {
    config = {
     codec: codecBox.value,
     iceServers: [
       {"username": turnUsername.value, "credential": turnCredential.value, "url": turnUrl.value}
     ],
     iceTransportPolicy: relayBox.options[relayBox.selectedIndex].value
   }
  }

  localRTC = new Ion.RTC(local, config);
  remoteRTC = new Ion.RTC(remote, config);

  let trackEvent;

  localRTC.join(sid, uid);

  remoteRTC.ontrackevent = function (ev) {
    console.log("[ontrackevent]: \nuid = ", ev.uid, " \nstate = ", ev.state, ", \ntracks = ", JSON.stringify(ev.tracks));
    if (!trackEvent) {
      trackEvent = ev;
    }
    remoteData.innerHTML = remoteData.innerHTML + JSON.stringify(ev) + '\n';
  };

  remoteRTC.ondatachannel = ({ channel }) => {
    console.log("[ondatachannel] channel=", channel)
    channel.onmessage = ({ data }) => {
      remoteData.innerHTML = remoteData.innerHTML + JSON.stringify(data) + '\n';
    };
  };

  remoteRTC.join(sid, "remote-user");

  remoteRTC.ontrack = (track, stream) => {
    if (track.kind === "video") {
      remoteStream = stream;
      remoteVideo.srcObject = stream;
      remoteVideo.autoplay = true;
      remoteVideo.muted = true;
      getStats();

      document.querySelectorAll(".controls")
        .forEach((el) => (el.style.display = "block"));
      if (simulcast) {
        document.getElementById("simulcast-controls").style.display =
          "block";
      } else {
        document.getElementById("simple-controls").style.display = "block";
      }
    }
  };

  joinBtn.removeAttribute('disabled');
  connectBtn.disabled = "true";

}

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
    });
    

}

const sendMsg = () => {
    if (!room) {
        alert('join room first!', '', {
        confirmButtonText: 'OK',
      });
      return
    }
    const payload = new Map();
    payload.set('msg', localData.value);
    console.log("[sendMsg]: sid=", sid, "from=", uid, "to=", "all", "payload=", payload);
    room.message(sid, "sender", "all", 'Map', payload);
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

remoteVideo.addEventListener("loadedmetadata", function () {
  sizeTag.innerHTML = `${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`;
});

remoteVideo.onresize = function () {
  sizeTag.innerHTML = `${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`;
};

let localStream;
const start = (sc) => {
  simulcast = sc;

  publishSBtn.disabled = "true";
  publishBtn.disabled = "true";

  Ion.LocalStream.getUserMedia({
    resolution: resolutionBox.options[resolutionBox.selectedIndex].value,
    codec:codecBox.options[codecBox.selectedIndex].value,
    simulcast: sc,
    audio: true,
  })
    .then((media) => {
      localStream = media;
      localVideo.srcObject = media;
      localVideo.autoplay = true;
      localVideo.controls = true;
      localVideo.muted = true;
      // joinBtns.style.display = "none";
      localRTC.publish(media);
      localDataChannel = localRTC.createDataChannel("data");
    })
    .catch(console.error);
};

const send = () => {
  if (!localDataChannel) {
      alert('click "start" first!', '', {
      confirmButtonText: 'OK',
    });
    return
  }
  if (localDataChannel.readyState === "open") {
    localDataChannel.send(localData.value);
  }
};

const api = {
  streamId: "",
  video: "high",
  audio: true,
};

const controlRemoteVideo = (radio) => {
  remoteStream.preferLayer(radio.value);

  // update ui
  api.streamId = remoteStream.id;
  api.video = radio.value;
  const str = JSON.stringify(api, null, 2);
  document.getElementById("api").innerHTML = syntaxHighlight(str);
};

const controlRemoteAudio = (radio) => {
  if (radio.value === "true") {
    remoteStream.mute("audio");
  } else {
    remoteStream.unmute("audio");
  }

  // update ui
  api.streamId = remoteStream.id;
  api.audio = radio.value === "true";
  const str = JSON.stringify(api, null, 2);
  document.getElementById("api").innerHTML = syntaxHighlight(str);
};

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

getStats = () => {
  let bytesPrev;
  let timestampPrev;
  setInterval(() => {
    remoteRTC.getSubStats(null).then((results) => {
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

        let rtt;
        if (report.type === "candidate-pair") {
          const rtt = report.currentRoundTripTime;
          if (rtt) {
            rttTag.innerHTML = `${rtt} s RTT`;
          }
        }
      });
    });
  }, 1000);
};
