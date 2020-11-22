const base_url = "http://" + window.location.hostname + "/api";
const ws_url = "ws://" + window.location.hostname + "/";

async function createUser(username) {
  return new Promise(function (resolve, reject) {
    var http = new XMLHttpRequest();
    var url = base_url + "/user/create";
    var params = "name=\"" + username + "\"";
    http.open('POST', url, true);
    http.setRequestHeader('Content-type', 'application/x-www-form-urlencoded');
    http.onload = function () {
      if (http.status == 200) {
        console.log(http.responseText);
        var data = JSON.parse(http.responseText);
        resolve(data)
      } else {
        reject()
      }
    }
    http.send(params);
  })
}

class Room {
  constructor(roomId) {
    this.roomId = roomId
    this.sessionId = ""
    this.pc = null
    this.hasRemoteOffer = false
    this.remoteICE = []
    this.socket = null
    this.localICE = []
  }
  getId() {
    return this.roomId;
  }
  async join(clientId, trackCB) {
    console.log(this);
    return new Promise(async (resolve, reject) => {
      this.socket = io(ws_url, {
        transports: ['websocket']
      });

      this.socket.on("connect", async () => {
        // Prepare for call
        const iceServers = await getICEServersConfiguration();
        this.pc = new RTCPeerConnection({
          iceServers: iceServers
        });

        this.pc.oniceconnectionstatechange = e => {
          console.log(e)
          console.log(this.pc.iceConnectionState)
          if (this.pc.iceConnectionState == "connected") {
            resolve();
          } else if (this.pc.iceConnectionState == "failed") {
            reject();
          }
        }

        this.pc.ontrack = function (event) {
          console.log("New track");
          console.log(event)
          trackCB(event.track, event.streams[0])
        }

        this.pc.onicecandidate = async (event) => {
          if (event.candidate !== null) {
            console.log("New candidate");
            console.log(event.candidate);
            let cand = btoa(JSON.stringify(event.candidate));
            if (this.hasRemoteOffer) {
              this.socket.emit("exchange_ice", JSON.stringify({
                "roomId": this.roomId,
                "uid": clientId,
                "ice": cand
              }));
            } else {
              this.localICE.push(cand);
            }
          } else {
            console.log("Finished gathering candidates");
          }
        }

        this.socket.on("session_start", (msg) => {
          console.log("Session started");
          let data = JSON.parse(msg);
          this.sessionId = data.sessionId;
        });
        this.socket.on("exchange_ice", async (msg) => {
          console.log("New remote ICE candidate");
          var enc = new TextDecoder("utf-8");
          var ice = JSON.parse(atob(enc.decode(msg)));
          console.log(ice);
          if (this.hasRemoteOffer) {
            this.pc.addIceCandidate(ice).catch(err => {
              console.warn("Failed to add remote ICE candidate");
              console.warn(err);
            });
          } else {
            this.remoteICE.push(ice);
          }
        });

        this.socket.on("remote_offer", async (msg) => {
          console.log("Accepting remote offer")
          let enc = new TextDecoder("utf-8");
          let data = JSON.parse(enc.decode(msg))
          let descr = JSON.parse(atob(data.data));
          let rtcDescr = new RTCSessionDescription(descr);
          await this.pc.setRemoteDescription(rtcDescr);
          const answer = await this.pc.createAnswer();
          await this.pc.setLocalDescription(answer);
          const offer = btoa(JSON.stringify(answer));
          this.socket.emit('peer_answer', JSON.stringify({
              "roomId": this.roomId,
              "uid": this.clientId,
              "offer": offer,
              "sessionId": this.sessionId
          }));
          this.hasRemoteOffer = true;
          this.localICE.forEach(async (cand) => {
            this.socket.emit("exchange_ice", JSON.stringify({
              "roomId": this.roomId,
              "uid": clientId,
              "ice": cand
            }));
          });
          this.remoteICE.forEach(async (ice) => {
            this.pc.addIceCandidate(ice).catch(err => {
              console.warn("Failed to add remote ICE candidate");
              console.warn(err);
            });
          })
        });

        // Initiate connection

        const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
        console.log("Created stream");
        stream.getTracks().forEach((track) => {
          this.pc.addTrack(track, stream)
          trackCB(track, new MediaStream([track]));
        });
        this.pc.addTransceiver("video");
        this.pc.addTransceiver("video");
        this.pc.addTransceiver("video");
        this.pc.addTransceiver("video");
        this.socket.emit('join_room', JSON.stringify({
          "roomId": this.roomId,
          "uid": clientId,
          "sessionId": this.sessionId
        }));
      });
    });
    // return new Promise(async (resolve, reject) => {
    // console.log(this);
    //   this.socket = io(ws_url, {
    //     transports: ['websocket']
    //   });

    //   this.socket.on("connect", async () => {
    //     console.log("Socket connected");
    //     this.socket.emit("handshake", JSON.stringify(
    //       {
    //         "roomId": this.roomId,
    //         "uid": clientId
    //       }
    //     ));

    //     const iceServers = await getICEServersConfiguration();
    //     this.pc = new RTCPeerConnection({
    //       iceServers: iceServers
    //     });

    // this.pc.oniceconnectionstatechange = e => {
    //   console.log(e)
    //   console.log(this.pc.iceConnectionState)
    //   if (this.pc.iceConnectionState == "connected") {
    //     resolve();
    //   } else if (this.pc.iceConnectionState == "failed") {
    //     reject();
    //   }
    //     }

    //     this.pc.onnegotiationneeded = () => {
    //       console.log("negotiation needed");
    //     }

    //     this.pc.ontrack = function (event) {
    //       console.log("New track");
    //       console.log(event)
    //       trackCB(event.track, event.streams[0])
    //     }

    //     this.pc.onicecandidate = async (event) => {
    //       if (event.candidate !== null) {
    //         console.log("New candidate");
    //         console.log(event.candidate);
    //         let cand = btoa(JSON.stringify(event.candidate));
    //         if (this.handshakeOk) {
    //           this.socket.emit("exchange_ice", JSON.stringify({
    //             "roomId": this.roomId,
    //             "uid": clientId,
    //             "ice": cand
    //           }));
    //         } else {
    //           this.localICE.push(cand);
    //         }
    //       } else {
    //         console.log("Finished gathering candidates");
    //       }
    //     }

    //     this.socket.on('exchange_ice', (msg) => {
    //       console.log("New remote ICE candidate");
    //       var enc = new TextDecoder("utf-8");
    //       var ice = JSON.parse(atob(enc.decode(msg)));
    //       console.log(ice);
    //       if (this.hasRemoteOffer) {
    //         this.pc.addIceCandidate(ice).catch(err => {
    //           console.warn("Failed to add remote ICE candidate");
    //           console.warn(err);
    //         });
    //       } else {
    //         this.remoteICE.push(ice);
    //       }
    //     });

    //     this.socket.on('handshake_ok', async (msg) => {
    //       this.handshakeOk = true;
    //       this.localICE.forEach((cand) => {
    //         this.socket.emit("exchange_ice", JSON.stringify({
    //           "roomId": this.roomId,
    //           "uid": clientId,
    //           "ice": cand
    //         }));
    //       });
    //     });

    //     this.socket.on('remote_offer', async (msg) => {
    //       console.log("Got offer");
    //       var enc = new TextDecoder("utf-8");
    //       var data = JSON.parse(enc.decode(msg))
    //       var descr = JSON.parse(atob(data.data));
    //       console.log(descr);
    //       var rtcDescr = new RTCSessionDescription(descr);
    //       this.pc.setRemoteDescription(rtcDescr).catch(err => {
    //         console.warn("Failed to set remote description");
    //         console.warn(err)
    //       })
    //       this.hasRemoteOffer = true;
    //       this.remoteICE.forEach((ice) => this.pc.addIceCandidate(ice));
    //     });

    //     try {
    //       const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
    //       console.log("Created stream");
    //       stream.getTracks().forEach((track) => {
    //         this.pc.addTrack(track, stream)
    //         trackCB(track, new MediaStream([track]));
    //       });
    //       this.pc.addTransceiver("video");
    //       this.pc.addTransceiver("video");
    //       this.pc.addTransceiver("video");
    //       this.pc.addTransceiver("video");
    //       this.socket.emit('join_room', JSON.stringify({
    //         "roomId": this.roomId,
    //         "uid": clientId
    //       }));
    //     } catch (err) {
    //       console.log(err);
    //     }
    //   });
    //   this.socket.on('session_start', async (msg) => {
    //     await this.pc.setLocalDescription(await this.pc.createOffer());
    //     let offer = btoa(JSON.stringify(this.pc.localDescription));
    //     let data = JSON.parse(msg);
    //     console.log(this.pc.localDescription)
    //     console.log("Local offer");
    //     console.log(offer);
    //     this.socket.emit('exchange_offer', JSON.stringify({
    //         "roomId": this.roomId,
    //         "uid": clientId,
    //         "offer": offer,
    //         "sessionId": data.sessionId
    //     }));
    //     this.sessionId = data.sessionId;
    //     console.log("Sent offer");
    //   });
    //   this.socket.connect();
    //   console.log(this);
    // });
  }

  async leave(clientId) {
    return new Promise((resolve, reject) => {
      this.pc.close();
      this.socket.emit("leave_room", JSON.stringify({
        "roomId": this.roomId,
        "uid": clientId,
        "sessionId": this.sessionId
      }))
      this.socket.close();
      resolve();
    });
  }
};

function createRoomWithId(roomId) {
  return new Room(roomId);
}

async function createRoom(ownerId) {
  return new Promise(function (resolve, reject) {
    var http = new XMLHttpRequest();
    var url = base_url + "/rooms/create";
    var params = "uid=\"" + ownerId + "\"";
    http.open('POST', url, true);
    http.onload = function () {
      if (http.status == 200) {
        console.log(http.responseText);
        var data = JSON.parse(http.responseText);
        resolve(createRoomWithId(data.id))
      } else {
        reject()
      }
    }
    http.send(params);
  })
}

async function getICEServersConfiguration() {
  return new Promise(function (resolve, reject) {
    var http = new XMLHttpRequest();
    var url = base_url + "/ice_servers";
    http.open('GET', url, true);
    http.onload = function () {
      if (http.status == 200) {
        console.log(http.responseText);
        var data = JSON.parse(http.responseText);
        resolve(data);
      } else {
        reject();
      }
    }
    http.send(null);
  });
}
