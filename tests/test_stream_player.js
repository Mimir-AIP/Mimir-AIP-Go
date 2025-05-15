import { StreamPlayer } from '../src/Plugins/WebInterface/static/js/stream-player.js';

describe('StreamPlayer', () => {
  let player;
  
  beforeEach(() => {
    player = new StreamPlayer();
  });

  test('should initialize player', () => {
    expect(player.playerElement).toBeNull();
    player.init();
    expect(player.playerElement).not.toBeNull();
  });

  test('should handle stream URL', () => {
    const testUrl = "http://test.stream/feed";
    player.handleStream(testUrl);
    expect(player.currentStream).toBe(testUrl);
  });
});