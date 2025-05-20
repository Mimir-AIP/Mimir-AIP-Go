/**
 * StreamPlayer component for handling live stream playback
 * Integrates with WebInterface's messaging system
 */

class StreamPlayer {
    constructor(options = {}) {
        // Configuration
        this.targetElementId = options.targetElementId || 'stream-player';
        this.autoplay = options.autoplay !== false;
        this.muted = options.muted !== false;
        this.controls = options.controls !== false;
        this.pipelineId = options.pipelineId || null;
        
        // Player state
        this.player = null;
        this.streamUrl = null;
        this.isPlaying = false;
        
        // Initialize
        this._createPlayerElement();
    }
    
    _createPlayerElement() {
        // Create video element container
        this.container = document.getElementById(this.targetElementId);
        if (!this.container) {
            console.error(`Target element not found: ${this.targetElementId}`);
            return;
        }
        
        // Create video element
        this.videoElement = document.createElement('video');
        this.videoElement.controls = this.controls;
        this.videoElement.autoplay = this.autoplay;
        this.videoElement.muted = this.muted;
        this.videoElement.style.width = '100%';
        
        this.container.appendChild(this.videoElement);
    }
    
    loadStream(streamUrl) {
        this.streamUrl = streamUrl;
        
        if (this._isHlsStream(streamUrl)) {
            this._loadHlsStream(streamUrl);
        } else {
            // Fallback to native video element
            this.videoElement.src = streamUrl;
        }
    }
    
    _isHlsStream(url) {
        return url.includes('.m3u8');
    }
    
    _loadHlsStream(url) {
        if (typeof Hls === 'undefined') {
            console.error('Hls.js not loaded - using native playback');
            this.videoElement.src = url;
            return;
        }
        
        // Initialize HLS.js
        if (this.player) {
            this.player.destroy();
        }
        
        this.player = new Hls({
            maxBufferLength: 30,
            maxMaxBufferLength: 600
        });
        
        this.player.loadSource(url);
        this.player.attachMedia(this.videoElement);
        
        this.player.on(Hls.Events.MANIFEST_PARSED, () => {
            this.isPlaying = true;
        });
        
        this.player.on(Hls.Events.ERROR, (event, data) => {
            console.error('HLS Error:', data);
            if (data.fatal) {
                this._handleFatalError();
            }
        });
    }
    
    _handleFatalError() {
        if (this.player) {
            this.player.destroy();
            this.player = null;
        }
        // Attempt native playback as fallback
        if (this.streamUrl && this.videoElement.canPlayType('application/vnd.apple.mpegurl')) {
            this.videoElement.src = this.streamUrl;
        }
    }
    
    destroy() {
        if (this.player) {
            this.player.destroy();
        }
        if (this.videoElement && this.videoElement.parentNode) {
            this.videoElement.parentNode.removeChild(this.videoElement);
        }
    }
}

// Export for WebInterface integration
if (typeof module !== 'undefined' && module.exports) {
    module.exports = StreamPlayer;
}

// Global handler for stream control commands from pipeline
window.handleStreamControl = function(command, data) {
    const player = window.currentStreamPlayer;
    if (!player) return;

    switch (command) {
        case 'load':
            player.loadStream(data.url);
            break;
        case 'play':
            player.videoElement.play();
            break;
        case 'pause':
            player.videoElement.pause();
            break;
        case 'stop':
            player.videoElement.pause();
            player.videoElement.currentTime = 0;
            break;
        case 'mute':
            player.videoElement.muted = true;
            break;
        case 'unmute':
            player.videoElement.muted = false;
            break;
    }
    
    // Send status update back to pipeline
    if (player.pipelineId) {
        const status = {
            type: 'stream_status',
            pipelineId: player.pipelineId,
            state: player.videoElement.paused ? 'paused' : 'playing',
            currentTime: player.videoElement.currentTime,
            muted: player.videoElement.muted
        };
        window.app.sendPipelineInput(status);
    }
};