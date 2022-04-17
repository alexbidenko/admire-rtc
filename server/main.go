package main

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	rtcpPLIInterval = time.Second * 3
)

type Sdp struct {
	Sdp string
}

func main() {
	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"POST"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	peerConnectionMap := make(map[string]map[string]chan *webrtc.Track)

	m := webrtc.MediaEngine{}

	m.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	router.POST("/webrtc/sdp/m/:meetingId/c/:userID/p/:peerId/s/:isSender", func(c *gin.Context) {
		isSender, _ := strconv.ParseBool(c.Param("isSender"))
		userID := c.Param("userID")
		peerID := c.Param("peerId")

		var session Sdp
		if err := c.ShouldBindJSON(&session); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		offer := webrtc.SessionDescription{}
		Decode(session.Sdp, &offer)

		peerConnection, err := api.NewPeerConnection(peerConnectionConfig)
		if err != nil {
			log.Fatal(err)
		}
		if !isSender {
			receiveTrack(peerConnection, peerConnectionMap, peerID)
		} else {
			err = createTrack(peerConnection, peerConnectionMap, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		peerConnection.SetRemoteDescription(offer)

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Fatal(err)
		}

		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, Sdp{Sdp: Encode(answer)})
	})

	router.Run(":8080")
}

func receiveTrack(
	peerConnection *webrtc.PeerConnection,
	peerConnectionMap map[string]map[string]chan *webrtc.Track,
	peerID string,
) {
	if _, ok := peerConnectionMap[peerID]; !ok {
		peerConnectionMap[peerID] = map[string]chan *webrtc.Track{
			"audio": make(chan *webrtc.Track, 1),
			"video": make(chan *webrtc.Track, 1),
		}
	}
	localTrackAudio := <-peerConnectionMap[peerID]["audio"]
	peerConnection.AddTrack(localTrackAudio)
	localTrackVideo := <-peerConnectionMap[peerID]["video"]
	peerConnection.AddTrack(localTrackVideo)
}

func createTrack(peerConnection *webrtc.PeerConnection,
	peerConnectionMap map[string]map[string]chan *webrtc.Track,
	currentUserID string) error {

	peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)
	peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo)

	peerConnection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(rtcpPLIInterval)
			for range ticker.C {
				if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}}); rtcpSendErr != nil {
					fmt.Println(rtcpSendErr)
				}
			}
		}()

		localTrack, newTrackErr := peerConnection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")
		if newTrackErr != nil {
			log.Fatal(newTrackErr)
		}

		localTrackChan := make(chan *webrtc.Track, 1)
		localTrackChan <- localTrack
		if existingChan, ok := peerConnectionMap[currentUserID]; ok {
			existingChan[remoteTrack.Kind().String()] <- localTrack
		} else {
			peerConnectionMap[currentUserID] = map[string]chan *webrtc.Track{
				localTrack.Kind().String(): localTrackChan,
			}
		}

		rtpBuf := make([]byte, 1400)
		for {
			i, readErr := remoteTrack.Read(rtpBuf)
			if readErr != nil {
				log.Fatal(readErr)
			}

			if _, err := localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
				log.Fatal(err)
			}
		}
	})
	return nil
}
