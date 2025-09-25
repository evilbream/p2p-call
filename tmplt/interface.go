package tmplt

var HtmlPage = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Audio WebSocket</title>
	<style>
		body {
			font-family: monospace;
			background: white;
			color: black;
			margin: 40px;
			line-height: 1.6;
		}
		button {
			background: white;
			color: black;
			border: 2px solid black;
			padding: 10px 20px;
			margin: 10px;
			cursor: pointer;
			font-family: monospace;
		}
		button:hover {
			background: black;
			color: white;
		}
		button:disabled {
			opacity: 0.5;
			cursor: not-allowed;
		}
		#status {
			margin: 20px 0;
			padding: 10px;
			border: 1px solid black;
		}
	</style>
</head>
<body>
	<h1>Audio WebSocket Interface</h1>
	
	<div>
		<button id="startRecord">Speak</button>
		<button id="stopRecord" disabled>Mute</button>
	</div>
	
	<div id="status">Status: Ready</div>
	
	<script>
		let ws;
		let mediaRecorder;
		let audioContext;
		let stream;
		let workletNode;
		let receivedAudioBuffer = [];
		let isPlaying = false;
		
		const status = document.getElementById('status');
		const startBtn = document.getElementById('startRecord');
		const stopBtn = document.getElementById('stopRecord');
		
		// WebSocket connection
		function connectWebSocket() {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			const wsUrl = protocol + '//' + window.location.host + '/ws';
			ws = new WebSocket(wsUrl);
			ws.binaryType = 'arraybuffer'; // Important: set binary type
			
			ws.onopen = () => {
				status.textContent = 'Status: Connected';
			};
			
			ws.onmessage = async (event) => {
				console.log('Received message type:', typeof event.data, event.data);
				
				let arrayBuffer;
				if (event.data instanceof ArrayBuffer) {
					arrayBuffer = event.data;
				} else if (event.data instanceof Blob) {
					arrayBuffer = await event.data.arrayBuffer();
				} else {
					console.log('Unknown data type, cannot process');
					return;
				}
				
				const pcmData = new Float32Array(arrayBuffer);
				console.log('Received audio data:', pcmData.length, 'samples');
				
				// Immediately play received audio
				if (!audioContext) {
					audioContext = new AudioContext();
					console.log('Created new audio context');
				}
				
				// Resume audio context if suspended
				if (audioContext.state === 'suspended') {
					await audioContext.resume();
					console.log('Resumed audio context');
				}
				
				const audioBuffer = audioContext.createBuffer(1, pcmData.length, 44100);
				audioBuffer.getChannelData(0).set(pcmData);
				
				const source = audioContext.createBufferSource();
				source.buffer = audioBuffer;
				
				// Добавляем усилитель громкости
				const gainNode = audioContext.createGain();
				gainNode.gain.value = 3.0; // Увеличиваем громкость в 3 раза
				
				source.connect(gainNode);
				gainNode.connect(audioContext.destination);
				source.start();
				
				console.log('Playing audio buffer with 3x gain');
				status.textContent = 'Status: Playing received audio (' + pcmData.length + ' samples)';
			};
			
			ws.onclose = () => {
				status.textContent = 'Status: Disconnected';
				setTimeout(connectWebSocket, 3000);
			};
			
			ws.onerror = (error) => {
				status.textContent = 'Status: Error - ' + error;
			};
		}
		
		// Initialize audio context
		async function initAudio() {
			audioContext = new AudioContext();
			
			try {
				stream = await navigator.mediaDevices.getUserMedia({ 
					audio: {
						sampleRate: 44100,
						channelCount: 1,
						echoCancellation: false,
						noiseSuppression: false
					} 
				});
			} catch (err) {
				status.textContent = 'Status: Microphone access denied';
				return;
			}
		}
		
		// Start recording
		startBtn.onclick = async () => {
			if (!audioContext) {
				await initAudio();
				// Resume audio context if suspended
				if (audioContext.state === 'suspended') {
					await audioContext.resume();
				}
			}
			if (!stream) return;
			
			const source = audioContext.createMediaStreamSource(stream);
			
			// Create ScriptProcessor for PCM data
			workletNode = audioContext.createScriptProcessor(4096, 1, 1);
			
			workletNode.onaudioprocess = (event) => {
				const inputBuffer = event.inputBuffer.getChannelData(0);
				const pcmData = new Float32Array(inputBuffer);
				
				if (ws && ws.readyState === WebSocket.OPEN) {
					console.log('Sending audio data:', pcmData.length, 'samples');
					ws.send(pcmData.buffer);
				}
			};
			
			source.connect(workletNode);
			workletNode.connect(audioContext.destination);
			
			startBtn.disabled = true;
			stopBtn.disabled = false;
			status.textContent = 'Status: Recording...';
		};
		
		// Stop recording
		stopBtn.onclick = () => {
			if (workletNode) {
				workletNode.disconnect();
				workletNode = null;
			}
			
			startBtn.disabled = false;
			stopBtn.disabled = true;
			status.textContent = 'Status: Recording stopped';
		};
		
		// Connect on page load
		connectWebSocket();
	</script>
</body>
</html>`
