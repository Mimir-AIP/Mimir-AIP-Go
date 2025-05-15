import { ChatComponent } from '../src/Plugins/WebInterface/static/js/llm-chat.js';

describe('LLMChat', () => {
  let chat;
  
  beforeEach(() => {
    chat = new ChatComponent();
  });

  test('should format messages correctly', () => {
    const testMsg = "Hello";
    const formatted = chat._formatMessage(testMsg);
    expect(formatted).toHaveProperty('timestamp');
    expect(formatted.content).toBe(testMsg);
  });

  test('should handle send message', () => {
    const mockSend = jest.fn();
    chat.sendMessage = mockSend;
    chat.handleSend("Test message");
    expect(mockSend).toHaveBeenCalled();
  });
});