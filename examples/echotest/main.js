const localVideo = document.getElementById("local-video");
const remoteVideo = document.getElementById("remote-video");
const localData = document.getElementById("local-data");
const remoteData = document.getElementById("remote-data");
const sizeTag = document.getElementById("size-tag");
const brTag = document.getElementById("br-tag");
const codecBox = document.getElementById("select-box1");
const resolutionBox = document.getElementById("select-box2");
let simulcast = false;
let localDataChannel;

let url = 'http://localhost:5551';
let sid = 'ion';
let uid = "echo-example";

let room;
const joinRoom = async () => {
    console.log("[joinRoom]: sid="+sid+" uid=", uid)
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
    
    function Uint8ArrayToString(fileData){
      var dataString = "";
      for (var i = 0; i < fileData.byteLength; i++) {
        dataString += String.fromCharCode(fileData[i]);
      }
      return dataString;
    }

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
    
    //room.connect();
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
             console.log('[joinRoom] result: success ' + result?.success + ', room info: ' + JSON.stringify(result?.room));
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
    console.log("[sendMsg]: sid=", sid, "from=", 'sender', "to=", uid, "payload=", payload);
    room.message(sid,'sender', uid, 'Map', payload);
}


const leaveRoom = () => {
    console.log("[leaveRoom]: sid="+sid+" uid=", uid)
    room.leave(sid, uid);
}

remoteVideo.addEventListener("loadedmetadata", function () {
  sizeTag.innerHTML = `${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`;
});

remoteVideo.onresize = function () {
  sizeTag.innerHTML = `${remoteVideo.videoWidth}x${remoteVideo.videoHeight}`;
};

/* eslint-env browser */
const joinBtns = document.getElementById("start-btns");

const local = new Ion.Connector(url);
const remote = new Ion.Connector(url);

local.onopen = function (service) {
  console.log("[onopen]: service = ", service.name);
};

local.onclose = function (service, err) {
  console.log("[onclose]: service = ", service.name, ", err = ", JSON.stringify(err.detail));
};

const localRTC = new Ion.RTC(local);
const remoteRTC = new Ion.RTC(remote);

localRTC.join(sid, uid);

remoteRTC.ontrackevent = function (ev) {
  console.log("[ontrackevent]: \nuid = ", ev.uid, " \nstate = ", ev.state, ", \ntracks = ", JSON.stringify(ev.tracks));
  event = ev;
  remoteData.innerHTML = remoteData.innerHTML + JSON.stringify(ev) + '\n';
};

remoteRTC.ondatachannel = ({ channel }) => {
  console.log("[ondatachannel] channel=", channel)
  channel.onmessage = ({ data }) => {
    remoteData.innerHTML = remoteData.innerHTML + JSON.stringify(data) + '\n';
  };
};

remoteRTC.join(sid, "echo-remote");

let localStream;
const start = (sc) => {
  simulcast = sc;
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
      joinBtns.style.display = "none";
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

let remoteStream;
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
      });
    });
  }, 1000);
};
