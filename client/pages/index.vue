<script setup lang="ts">
import {onMounted} from "vue";
import axios from "axios";
import {useRoute, useRouter} from "vue-router";

const router = useRouter();
const route = useRoute();

interface Sdp {
  Sdp: string;
}

let pcSender: RTCPeerConnection
let pcReceiver: RTCPeerConnection
const meetingId = route.query.meetingId as string;
const peerId = route.query.peerId as string;
const userId = route.query.userId as string;

if (!meetingId || !peerId || !userId) router.push('/join');

const startCall = () => {
  navigator.mediaDevices.getUserMedia({video: true, audio: true}).then((stream) =>{
    const senderVideo: any = document.getElementById('senderVideo');
    senderVideo.srcObject = stream;
    const tracks = stream.getTracks();
    for (let i = 0; i < tracks.length; i++) {
      pcSender.addTrack(stream.getTracks()[i]);
    }
    pcSender.addTransceiver('video')
    pcSender.addTransceiver('audio')
    pcSender.createOffer().then(d => pcSender.setLocalDescription(d))
  })
  pcSender.addEventListener('connectionstatechange', () => {
    if (pcSender.connectionState === 'connected') {
      console.log("horray!")
    }
  });

  pcReceiver.addTransceiver('video')
  pcReceiver.addTransceiver('audio')

  pcReceiver.createOffer().then(d => pcReceiver.setLocalDescription(d))

  pcReceiver.ontrack = function (event) {
    const receiverVideo: any = document.getElementById('receiverVideo');
    receiverVideo.srcObject = event.streams[0]
    receiverVideo.autoplay = true
    receiverVideo.controls = true
  }
}

onMounted(() => {
  pcSender = new RTCPeerConnection({
    iceServers: [
      {
        urls: 'stun:stun.l.google.com:19302'
      }
    ]
  })
  pcReceiver = new RTCPeerConnection({
    iceServers: [
      {
        urls: 'stun:stun.l.google.com:19302'
      }
    ]
  })

  pcSender.onicecandidate = event => {
    if (event.candidate === null) {
      axios.post<Sdp>(
          '/webrtc/sdp/m/' + meetingId + "/c/"+ userId + "/p/" + peerId + "/s/" + true,
          {"sdp" : btoa(JSON.stringify(pcSender.localDescription))},
      ).then(({data}) => {
        pcSender.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(data.Sdp))))
      })
    }
  }
  pcReceiver.onicecandidate = event => {
    if (event.candidate === null) {
      axios.post<Sdp>(
          '/webrtc/sdp/m/' + meetingId + "/c/"+ userId + "/p/" + peerId + "/s/" + false,
          {"sdp" : btoa(JSON.stringify(pcReceiver.localDescription))}
      ).then(({data}) => {
        pcReceiver.setRemoteDescription(new RTCSessionDescription(JSON.parse(atob(data.Sdp))))
      })
    }
  }

  startCall();
});
</script>

<template>
  <div>
    <video autoplay id="senderVideo" width="500" height="500" muted></video>
    <video autoplay id="receiverVideo" width="500" height="500"></video>
  </div>
</template>
