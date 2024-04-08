import { useRef, useState } from "react";
import { RTCClient } from "./Hooks/RTCClient";

const client = new RTCClient("52.14.127.124:8080", "stun:stun1.l.google.com:19302", "stun:stun2.l.google.com:19302")
client.FindDataChannel("test").then(chan => {
	let rebuilt = ""
	chan.onmessage = m => {
		if (m.data == "END_OF_FRAME") {
			console.log(rebuilt)
			triggerer(rebuilt)
			rebuilt = ""
			chan.send("")
		} else {
			rebuilt += m.data
		}
	}
})

let triggerer = (m: string) => {}

export default function App() {
	const [imgSrc, setSrc] = useState<string>("")

	triggerer = m => {
		setSrc("data:image/jpeg;base64,"+m)
	}
	
	return (
		<div style={{display:"flex", width:"100vw", height:"100vh", alignItems:"center", justifyContent:"center", flexDirection:"column"}}>
	  		<img src={imgSrc} style={{maxWidth:"100%", maxHeight:"100%"}} />
		</div>
 	);
}
