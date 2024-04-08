package RTC

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pion/webrtc/v3"
)

type RTCClient struct {
	SSEUnregisterURL string
	SSERegisterURL   string
	SSEListenURL     string
	SSECallURL       string

	channels map[string]*webrtc.DataChannel
	Conn     *webrtc.PeerConnection
	ice      []webrtc.ICECandidate
	id       string
}
type Request struct {
	Action        string                    `json:"action"`
	CallerID      string                    `json:"callerID"`
	Offer         webrtc.SessionDescription `json:"offer"`
	IceCandidates []webrtc.ICECandidate     `json:"iceCandidates"`
}

type Response struct {
	Action        string                    `json:"action"`
	CalleeID      string                    `json:"calleeID"`
	Answer        webrtc.SessionDescription `json:"answer"`
	IceCandidates []webrtc.ICECandidate     `json:"iceCandidates"`
}

func NewClient(SSEHost string, iceURLS ...string) (RTCClient, error) {
	this := RTCClient{}

	this.SSERegisterURL  = "http://" + SSEHost + "/register"
	this.SSEUnregisterURL= "http://" + SSEHost + "/unregister"
	this.SSEListenURL    = "http://" + SSEHost + "/listen"
	this.SSECallURL      = "http://" + SSEHost + "/call"

	peerConnection, err := webrtc.NewAPI().NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: iceURLS}},
	})
	this.Conn = peerConnection
	if err != nil {return RTCClient{}, err}
	
	this.channels = make(map[string]*webrtc.DataChannel)
	this.ice = []webrtc.ICECandidate{}
	this.id = ""

	// ice listeners
	this.Conn.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i != nil {
			this.ice = append(this.ice, *i)
		}
	})

	// would have ondatachannel event, but unnessecary cause handler doesnt get overwritten

	this.SendDataChannel("init") // doesnt get connected, establish ice-username fragment

	// sse register
	resp, err := http.Get(this.SSERegisterURL + "?id=BB")
	if err != nil {return RTCClient{}, err}
	data, err := io.ReadAll(resp.Body)
	if err != nil {return RTCClient{}, err}
	this.id = string(data)

	println("YOUR ID:", this.id)

	go this.signalingListen()

	go this.waitUnregister()

	return this, nil
}

func (c *RTCClient) waitUnregister() {
	// Create a channel to receive OS signals
	signalChannel := make(chan os.Signal, 1)

	// Notify the signal channel for interrupt signals (e.g., Ctrl+C)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	// Run a goroutine to handle the signal
	// Wait for the signal
	<-signalChannel

	http.Get(c.SSEUnregisterURL + "?id=" + c.id)

	// Terminate the program gracefully
	os.Exit(0)
}

func (c *RTCClient) signalingListen() {
	println("listening?????")
	client := http.Client{}

	// create request obj
	req, err := http.NewRequest("GET", c.SSEListenURL+"?id="+c.id, nil)
	if err != nil {
		println("signalingListen err 1:", err.Error())
		return
	}
	req.Header.Set("Accept", "text/event-stream")

	// send request
	resp, err := client.Do(req)
	if err != nil {
		println("signalingListen err 2:", err.Error())
		return
	}
	defer resp.Body.Close()

	println("sse open!!!")

	// Read SSE events from the response body
	reader := resp.Body
	for {
		// read bytes from sse
		bytes := make([]byte, 10000)
		_, err := reader.Read(bytes)
		if err != nil {
			println("signalingListen err 3:", err.Error())
			return
		}

		// trim and un nullify bytes
		cleaned := strings.Replace(strings.TrimSpace(strings.Split(string(bytes), "data: ")[1]), "\x00", "", -1)

		//println(cleaned)

		// read action param
		data := make(map[string]json.RawMessage)
		err = json.Unmarshal([]byte(cleaned), &data)
		if err != nil {
			println("signalingListen err 4:", err.Error())
			return
		}

		// figure out which action is needed to properly parse out the data
		switch strings.ReplaceAll(string(data["action"]), "\"", "") {
		case "receiveRequest":
			var req Request
			err = json.Unmarshal([]byte(cleaned), &req)
			if err != nil {
				println("signalingListen err 5:", err.Error())
				return
			}
			c.receiveRequestSendResponse(req)
		case "receiveResponse":
			var res Response
			err = json.Unmarshal([]byte(cleaned), &res)
			if err != nil {
				println("signalingListen err 6:", err.Error())
				return
			}
			c.receiveResponse(res)
		default:
			println("case not recognized")
		}
	}
}

func (c *RTCClient) Connect(calleeID string) {
	c.sendRequest(calleeID)
}

func (c *RTCClient) sendRequest(calleeID string) {
	println("creating offer...")

	// create local offer
	offer, err := c.Conn.CreateOffer(&webrtc.OfferOptions{})
	if err != nil {
		println("error", err.Error())
		return
	}
	err = c.Conn.SetLocalDescription(offer)
	if err != nil {
		println("error", err.Error())
		return
	}

	// call params
	json, err := json.Marshal(Request{
		Action:        "receiveRequest",
		CallerID:      c.id,
		Offer:         offer,
		IceCandidates: c.ice,
	})
	if err != nil {
		println("error", err.Error())
		return
	}

	// send a potiential call
	http.Post(c.SSECallURL+"?id="+calleeID, "text/plain", bytes.NewBuffer(json))

	println("sent offer")
}

// receive offer, create answer, and send it
func (c *RTCClient) receiveRequestSendResponse(data Request) {
	println("creating answer...")

	// take remote offer
	err := c.Conn.SetRemoteDescription(data.Offer)
	if err != nil {
		println("error", err.Error())
		return
	}

	// create local answer
	answer, err := c.Conn.CreateAnswer(&webrtc.AnswerOptions{})
	if err != nil {
		println("error", err.Error())
		return
	}
	err = c.Conn.SetLocalDescription(answer)
	if err != nil {
		println("error", err.Error())
		return
	}

	// add remote ice
	for _, ice := range data.IceCandidates {
		c.Conn.AddICECandidate(ice.ToJSON())
	}

	// call params
	json, err := json.Marshal(Response{
		Action:        "receiveResponse",
		CalleeID:      c.id,
		Answer:        answer,
		IceCandidates: c.ice,
	})
	if err != nil {
		println("error", err.Error())
		return
	}

	// send potential call
	http.Post(c.SSECallURL+"?id="+data.CallerID, "text/plain", bytes.NewBuffer(json))

	println("sent answer")
}

// receive response, try connect, if fail, run send request
func (c *RTCClient) receiveResponse(data Response) {
	println("receiving answer...")

	// take remote answer
	err := c.Conn.SetRemoteDescription(data.Answer)
	if err != nil {
		//println(data.IceCandidates)
		println("error (probably havent mad channels / tracks before this happened)", err.Error())
		//println("retrying...")
		//c.sendRequest(data.CalleeID)
		return
	}

	// add remote ice
	for _, ice := range data.IceCandidates {
		c.Conn.AddICECandidate(ice.ToJSON())
	}

	println("checking connection...")

	// add time to ensure connection
	time.Sleep(500 * time.Millisecond)

	println("connection:", c.Conn.ICEConnectionState().String(), "\n")

	// retry if not connected
	if c.Conn.ICEConnectionState().String() != "connected" {
		println("retrying...")
		c.sendRequest(data.CalleeID)
	}
}

func (c *RTCClient) FindDataChannel(name string) *webrtc.DataChannel {
	finder := make(chan *webrtc.DataChannel)

	c.Conn.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc.Label() == name {
			finder <- dc
		}
	})

	return <-finder
}

func (c *RTCClient) SendDataChannel(name string) (*webrtc.DataChannel, error) {
	return c.Conn.CreateDataChannel(name, &webrtc.DataChannelInit{})
}