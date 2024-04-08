type RTCRequest = {
	action        :string                    
	callerID      :string                    
	offer         :RTCSessionDescriptionInit 
	iceCandidates :RTCIceCandidate[]  
}

type RTCResponse  = {
	action        :string                    
	calleeID      :string                    
	answer        :RTCSessionDescriptionInit 
	iceCandidates :RTCIceCandidate[]  
}

export class RTCClient {
    SSERegisterURL: string;
    SSEUnregisterURL: string;
    SSEListenURL: string;
    SSECallURL: string;

    Conn: RTCPeerConnection;
    ice: RTCIceCandidate[];
    id: string;
    channelFinders:  Map<string, any>;

    constructor(SSEHost:string , ...servers: string[]) {
        this.SSERegisterURL  = "http://" + SSEHost + "/register"
        this.SSEUnregisterURL= "http://" + SSEHost + "/unregister"
        this.SSEListenURL    = "http://" + SSEHost + "/listen"
        this.SSECallURL      = "http://" + SSEHost + "/call"

        this.Conn = new RTCPeerConnection({
            iceServers: [{urls: servers}],  
        })

        this.channelFinders = new Map()
        this.ice = []
        this.id = ""

        this.Conn.onicecandidate = event => {
            if (event.candidate) {
                this.ice.push(event.candidate)
            }
        }

        this.Conn.ondatachannel = event => {
            this.channelFinders.forEach((v, k) => {
                v(event)
            })
        }

        this.SendDataChannel("init") // doesnt get connected, establish ice-username fragment

        fetch(this.SSERegisterURL+"?id=AA").then(res => {
            res.text().then(text => {
                this.id = text

                console.log("YOUR ID:", this.id)

                this.signalingListen()
                
                this.waitUnregister()
            })
        })  
    }

    waitUnregister() {
        window.addEventListener('beforeunload', (event) => {
            event.preventDefault();
            navigator.sendBeacon(`${this.SSEUnregisterURL}?id=${this.id}`);
        });
    }

    signalingListen() {
        const eventSource = new EventSource(this.SSEListenURL+"?id="+this.id);
        eventSource.onerror = event => console.log("listen err",event)
        eventSource.onopen = event => console.log("listen open", event)

        eventSource.onmessage = event => {
            let parsed = JSON.parse(event.data)
            switch(parsed.action) {
                case "receiveRequest": this.receiveRequestSendResponse(parsed as RTCRequest); break
                case "receiveResponse": this.receiveResponse(parsed as RTCResponse); break
            }
        }
    }

    // create offer and send it
    async sendRequest(calleeID: string) {
        console.log("creating offer...")

        // create local offer
        const offer = await this.Conn.createOffer()
        await this.Conn.setLocalDescription(offer)

        // send a potiential call
        fetch(this.SSECallURL+"?id="+calleeID, {
            method: 'POST',
            headers: {'Content-Type': 'text/plain'},
            body: JSON.stringify({
                action: "receiveRequest",
                callerID: this.id,
                offer: offer,
                iceCandidates: this.ice
            })
        })

        console.log("sent offer")
    }

    // receive offer, create answer, and send it
    async receiveRequestSendResponse(request: RTCRequest) {
        console.log("creating answer...")

        // take remote offer
        await this.Conn.setRemoteDescription(request.offer)

        // create local answer
        const answer = await this.Conn.createAnswer()
        await this.Conn.setLocalDescription(answer)

        // add remote ice
        request.iceCandidates.forEach(candidate => this.Conn?.addIceCandidate(candidate))

        // send a potiential call
        fetch(this.SSECallURL+"?id="+request.callerID, {
            method: 'POST',
            headers: {'Content-Type': 'text/plain'},
            body: JSON.stringify({
                action: "receiveResponse",
                calleeID: this.id,
                answer: answer,
                iceCandidates: this.ice
            })
        })

        console.log("sent answer")
    }

    // receive response, try connect, if fail, run send request
    async receiveResponse(response: RTCResponse) {
        console.log("receiving answer...")

        // take remote answer
        await this.Conn.setRemoteDescription(response.answer)

        // add remote ice
        response.iceCandidates.forEach(candidate => this.Conn?.addIceCandidate(candidate))

        console.log("checking connection...")

        // retry if not connected
        window.setTimeout(() => {
            if (this.Conn.iceConnectionState != "connected") {
                console.log("retrying...")
                this.sendRequest(response.calleeID)// send to sender
            }
        }, 500)
    }
    
    FindDataChannel(name:string): Promise<RTCDataChannel> {
        console.log("FINDING CHANNEL",name)

        return new Promise((res) => {
            if (!this.channelFinders.has(name)) { // basically dont set the ondatachannel to a whole diff channel, loosing the finding ability of another
                this.channelFinders.set(name, (event:RTCDataChannelEvent) => {
                    if (event.channel.label == name) {
                        res(event.channel)
                        this.channelFinders.delete(name)
                    }
                })
            }
        })
    }

    SendDataChannel(name: string): RTCDataChannel {
        console.log("OPENING CHANNEL",name)
        const chan = this.Conn.createDataChannel(name)

        return chan
    }
}