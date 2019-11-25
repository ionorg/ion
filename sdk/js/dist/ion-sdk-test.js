var client = new Client();
var connected = false;
var published = false;
var streams = new Map();

window.onunload = function () {
    client.leave();
}

client.on('peer-join', (id, rid) => {
    showStatus('peer => ' + id + ', join!');
});

client.on('peer-leave', (id, rid) => {
    showStatus('peer => ' + id + ', leave!');
});

client.on('transport-open', function () {
    showStatus('transport open!');
    connected = true;
});

client.on('transport-closed', function () {
    showStatus('transport closed!');
    connected = false;
});

client.on('stream-add', async (id, rid) => {
    let stream = await client.subscribe(id);
    streams[id] = stream;
    insertVideoView('remote-video-container', id);
    stream.render(stream.uid);
});

client.on('stream-remove', async (id, rid) => {
    let stream = streams[id];
    removeVideoView(id);
    stream.stop();
    delete streams[id];
});

function insertVideoView(parentId, id) {
    let parentNode = document.getElementById(parentId);
    let element = document.createElement("div");
    element.id = id;
    parentNode.appendChild(element);
}

function removeVideoView(id) {
    let element = document.getElementById(id);
    element.parentNode.removeChild(element);
}

function showStatus(text) {
    var element = document.getElementById('status');
    element.value = text;
    console.log(text);
}

function onJoinBtnClick() {
    var element = document.getElementById('roomId');
    var roomId = element.value;
    if (roomId === '')
        return;
    showStatus('join to [' + roomId + ']');
    client.join(roomId);
}

async function onPublishBtnClick() {
    if (!connected) {
        alert('not connected to the server!');
        return;
    }
    if (published) {
        alert('already published!');
        return;
    }
    showStatus('start publish!');
    let stream = await client.publish({ codec: 'H264' });
    let id = stream.uid;
    insertVideoView('local-video-container', id);
    stream.render(id);
    published = true;
}

client.init();