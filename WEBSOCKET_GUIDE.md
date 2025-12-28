# WebSocket Integration Guide for Flutter

This guide covers WebSocket integration for the MomLaunchpad chat feature using Flutter.

## Overview

The chat backend uses **WebSocket** for real-time streaming responses from the AI. HTTP is only used for authentication, calendar CRUD, and admin operations.

**WebSocket endpoint:** `ws://your-domain.com/ws/chat?token={JWT_TOKEN}`

---

## Connection Setup

### 1. Add Dependencies

```yaml
# pubspec.yaml
dependencies:
  web_socket_channel: ^2.4.0
  provider: ^6.1.0  # For state management
```

### 2. Basic Connection (Dart)

```dart
import 'package:web_socket_channel/web_socket_channel.dart';

class ChatService {
  WebSocketChannel? _channel;
  
  Future<void> connect(String jwtToken) async {
    final uri = Uri.parse('ws://api.momlaunchpad.com/ws/chat?token=$jwtToken');
    
    _channel = WebSocketChannel.connect(uri);
    
    // Listen to messages
    _channel!.stream.listen(
      (message) => _handleMessage(message),
      onError: (error) => _handleError(error),
      onDone: () => _handleDisconnect(),
    );
  }
  
  void sendMessage(String content) {
    if (_channel != null) {
      _channel!.sink.add(jsonEncode({'content': content}));
    }
  }
  
  void dispose() {
    _channel?.sink.close();
  }
}
```

### 3. Authentication Flow

```dart
// 1. Login via HTTP to get JWT
final loginResponse = await http.post(
  Uri.parse('http://api.momlaunchpad.com/api/auth/login'),
  body: jsonEncode({'email': email, 'password': password}),
  headers: {'Content-Type': 'application/json'},
);

final token = jsonDecode(loginResponse.body)['token'];

// 2. Connect WebSocket with token
await chatService.connect(token);
```

---

## Message Protocol

### Outgoing Messages (Client → Server)

**Format:** JSON object with `content` field

```dart
// Send user message
final message = jsonEncode({
  'content': 'I\'m 14 weeks pregnant and feeling nauseous'
});
_channel.sink.add(message);
```

**Examples:**
```dart
// Pregnancy question
_channel.sink.add(jsonEncode({'content': 'When will my baby start kicking?'}));

// Symptom report
_channel.sink.add(jsonEncode({'content': 'I have severe headache'}));

// Small talk
_channel.sink.add(jsonEncode({'content': 'hello'}));

// Scheduling
_channel.sink.add(jsonEncode({'content': 'remind me to take vitamins'}));
```

### Incoming Messages (Server → Client)

The server sends **multiple message types**:

**Note:** Backend currently supports **English (en)**, **Spanish (es)**, and **French (fr)**.

#### 1. Streaming AI Response

```json
{
  "type": "message",
  "content": "It's great that you're asking about..."
}
```

**Important:** AI responses come in **chunks**. You must concatenate them:

```dart
String _currentResponse = '';

void _handleMessage(dynamic data) {
  final message = jsonDecode(data);
  
  switch (message['type']) {
    case 'message':
      // Append chunk to current response
      _currentResponse += message['content'];
      _updateUI(_currentResponse);
      break;
      
    case 'done':
      // Response complete
      _finalizeMessage(_currentResponse);
      _currentResponse = '';
      break;
      
    case 'calendar':
      final suggestionData = message['data'];
      _showCalendarSuggestion(
        title: suggestionData['title'],
        description: suggestionData['description'],
        suggestedTime: DateTime.parse(suggestionData['suggested_time']),
      );
      break;
      
    case 'error':
      _showError(message['message']);
      break;
  }
}
```

#### 2. Calendar Suggestion

```json
{
  "type": "calendar",
  "data": {
    "type": "reminder",
    "title": "Monitor symptom",
    "description": "Check in on nausea levels",
    "suggested_time": "2024-03-15T14:00:00Z"
  }
}
```

**UI Action:** Show a button/dialog asking user to confirm reminder creation. Use the structured `data` object to pre-fill reminder details.

#### 3. Response Complete

```json
{
  "type": "done"
}
```

**UI Action:** Stop loading indicator, finalize message bubble.

#### 4. Error Message

```json
{
  "type": "error",
  "message": "Rate limit exceeded. Please wait."
}
```

**UI Action:** Show error toast, disable send button temporarily.

---

## Rate Limiting

### Limits
- **10 messages per minute** per user
- **429 status** if exceeded

### Handling Rate Limits

```dart
class ChatService {
  DateTime? _lastMessageTime;
  int _messageCount = 0;
  
  bool canSendMessage() {
    final now = DateTime.now();
    
    // Reset counter every minute
    if (_lastMessageTime == null || 
        now.difference(_lastMessageTime!) > Duration(minutes: 1)) {
      _messageCount = 0;
      _lastMessageTime = now;
    }
    
    // Check limit
    return _messageCount < 10;
  }
  
  void sendMessage(String content) {
    if (!canSendMessage()) {
      _showError('Please wait before sending another message');
      return;
    }
    
    _channel!.sink.add(jsonEncode({'content': content}));
    _messageCount++;
  }
}
```

### UI Feedback

```dart
// Show remaining messages
Widget _buildSendButton() {
  final canSend = chatService.canSendMessage();
  
  return ElevatedButton(
    onPressed: canSend ? _sendMessage : null,
    child: Text(canSend ? 'Send' : 'Rate limited'),
  );
}
```

---

## Message Types & Expected Behavior

### Small Talk (Instant Response)

**Triggers:** `hello`, `hi`, `thanks`, `bye`, `how are you`

**Behavior:**
- No AI call (instant response)
- No memory saved
- Canned responses

**UI:** Don't show "typing..." indicator for small talk.

```dart
bool isSmallTalk(String content) {
  final smallTalkPatterns = ['hello', 'hi', 'thanks', 'bye', 'how are you'];
  final normalized = content.toLowerCase().trim();
  return smallTalkPatterns.any((pattern) => normalized.contains(pattern));
}

// In send function
if (isSmallTalk(content)) {
  _showTypingIndicator = false; // Instant response expected
}
```

### Pregnancy Questions (Streaming AI)

**Triggers:** Questions about pregnancy, baby development, health

**Behavior:**
- Streams response in chunks
- Saves to memory
- May suggest calendar reminders

**UI:** Show "typing..." indicator, then stream text as it arrives.

### Symptom Reports (High Priority)

**Triggers:** `I have...`, `I'm experiencing...`, `Is it normal to...`

**Behavior:**
- Urgent priority
- Always suggests calendar reminder
- Stores symptom facts

**UI:** Highlight calendar suggestions for symptoms.

---

## Error Handling

### Connection Errors

```dart
void _handleError(dynamic error) {
  print('WebSocket error: $error');
  
  // Show error UI
  _showError('Connection lost. Reconnecting...');
  
  // Attempt reconnection
  Future.delayed(Duration(seconds: 5), () {
    if (_shouldReconnect) {
      connect(_jwtToken);
    }
  });
}

void _handleDisconnect() {
  print('WebSocket disconnected');
  
  // Automatic reconnection
  if (_shouldReconnect) {
    Future.delayed(Duration(seconds: 3), () {
      connect(_jwtToken);
    });
  }
}
```

### Server Error Responses

```dart
void _handleServerError(Map<String, dynamic> message) {
  final errorMsg = message['message'] ?? 'Unknown error';
  
  switch (errorMsg) {
    case 'Rate limit exceeded':
      _showRateLimitDialog();
      break;
      
    case 'Invalid token':
      _handleTokenExpired();
      break;
      
    case 'DeepSeek API unavailable':
      _showFallbackMessage();
      break;
      
    default:
      _showError(errorMsg);
  }
}

void _showRateLimitDialog() {
  showDialog(
    context: context,
    builder: (context) => AlertDialog(
      title: Text('Slow down'),
      content: Text('You\'re sending messages too quickly. Please wait a moment.'),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text('OK'),
        ),
      ],
    ),
  );
}
```

### Token Expiration

```dart
void _handleTokenExpired() {
  // Clear local token
  await _authService.clearToken();
  
  // Navigate to login
  Navigator.pushReplacementNamed(context, '/login');
  
  // Show message
  _showError('Session expired. Please login again.');
}
```

---

## Complete Flutter Example

### ChatProvider (State Management)

```dart
import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'dart:convert';

class Message {
  final String id;
  final String content;
  final bool isUser;
  final DateTime timestamp;
  final bool isStreaming;
  
  Message({
    required this.id,
    required this.content,
    required this.isUser,
    required this.timestamp,
    this.isStreaming = false,
  });
}

class ChatProvider extends ChangeNotifier {
  WebSocketChannel? _channel;
  List<Message> _messages = [];
  bool _isConnected = false;
  String _currentResponse = '';
  
  List<Message> get messages => _messages;
  bool get isConnected => _isConnected;
  
  Future<void> connect(String token) async {
    try {
      final uri = Uri.parse('ws://api.momlaunchpad.com/ws/chat?token=$token');
      _channel = WebSocketChannel.connect(uri);
      _isConnected = true;
      notifyListeners();
      
      _channel!.stream.listen(
        _handleMessage,
        onError: _handleError,
        onDone: _handleDisconnect,
      );
    } catch (e) {
      _isConnected = false;
      notifyListeners();
      debugPrint('Connection error: $e');
    }
  }
  
  void sendMessage(String content) {
    if (!_isConnected || content.trim().isEmpty) return;
    
    // Add user message to UI
    _messages.add(Message(
      id: DateTime.now().toString(),
      content: content,
      isUser: true,
      timestamp: DateTime.now(),
    ));
    
    // Send to server
    _channel!.sink.add(jsonEncode({'content': content}));
    
    // Add empty AI message for streaming
    _messages.add(Message(
      id: 'ai_${DateTime.now()}',
      content: '',
      isUser: false,
      timestamp: DateTime.now(),
      isStreaming: true,
    ));
    
    _currentResponse = '';
    notifyListeners();
  }
  
  void _handleMessage(dynamic data) {
    try {
      final message = jsonDecode(data);
      
      switch (message['type']) {
        case 'message':
          // Append chunk
          _currentResponse += message['content'] ?? '';
          
          // Update last message
          if (_messages.isNotEmpty && !_messages.last.isUser) {
            _messages.last = Message(
              id: _messages.last.id,
              content: _currentResponse,
              isUser: false,
              timestamp: _messages.last.timestamp,
              isStreaming: true,
            );
          }
          notifyListeners();
          break;
          
        case 'done':
          // Finalize message
          if (_messages.isNotEmpty && !_messages.last.isUser) {
            _messages.last = Message(
              id: _messages.last.id,
              content: _currentResponse,
              isUser: false,
              timestamp: _messages.last.timestamp,
              isStreaming: false,
            );
          }
          _currentResponse = '';
          notifyListeners();
          break;
          
        case 'calendar':
          // Show calendar suggestion (structured data)
          final data = message['data'];
          _showCalendarSuggestion(
            data['title'],
            data['description'],
            DateTime.parse(data['suggested_time']),
          );
          break;
          
        case 'error':
          // Show error
          debugPrint('Server error: ${message['message']}');
          break;
      }
    } catch (e) {
      debugPrint('Message parsing error: $e');
    }
  }
  
  void _handleError(dynamic error) {
    _isConnected = false;
    notifyListeners();
    debugPrint('WebSocket error: $error');
  }
  
  void _handleDisconnect() {
    _isConnected = false;
    notifyListeners();
    debugPrint('WebSocket disconnected');
  }
  
  void _showCalendarSuggestion(String title, String description, DateTime suggestedTime) {
    // Emit event for UI to show dialog
    debugPrint('Calendar suggestion: $title at $suggestedTime');
    debugPrint('Description: $description');
    // TODO: Show dialog with pre-filled reminder details
  }
  
  void dispose() {
    _channel?.sink.close();
    super.dispose();
  }
}
```

### ChatScreen UI

```dart
class ChatScreen extends StatefulWidget {
  @override
  _ChatScreenState createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final TextEditingController _controller = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  
  @override
  void initState() {
    super.initState();
    
    // Connect WebSocket
    final token = Provider.of<AuthProvider>(context, listen: false).token;
    Provider.of<ChatProvider>(context, listen: false).connect(token);
  }
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Chat'),
        actions: [
          Consumer<ChatProvider>(
            builder: (context, chat, child) {
              return Icon(
                chat.isConnected ? Icons.cloud_done : Icons.cloud_off,
                color: chat.isConnected ? Colors.green : Colors.red,
              );
            },
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: Consumer<ChatProvider>(
              builder: (context, chat, child) {
                return ListView.builder(
                  controller: _scrollController,
                  itemCount: chat.messages.length,
                  itemBuilder: (context, index) {
                    final message = chat.messages[index];
                    return _buildMessageBubble(message);
                  },
                );
              },
            ),
          ),
          _buildInputField(),
        ],
      ),
    );
  }
  
  Widget _buildMessageBubble(Message message) {
    return Align(
      alignment: message.isUser ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: EdgeInsets.all(8),
        padding: EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: message.isUser ? Colors.blue : Colors.grey[300],
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              message.content,
              style: TextStyle(
                color: message.isUser ? Colors.white : Colors.black,
              ),
            ),
            if (message.isStreaming)
              SizedBox(height: 4),
            if (message.isStreaming)
              SizedBox(
                width: 12,
                height: 12,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
          ],
        ),
      ),
    );
  }
  
  Widget _buildInputField() {
    return Container(
      padding: EdgeInsets.all(8),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _controller,
              decoration: InputDecoration(
                hintText: 'Type a message...',
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(24),
                ),
              ),
              maxLines: null,
            ),
          ),
          SizedBox(width: 8),
          Consumer<ChatProvider>(
            builder: (context, chat, child) {
              return IconButton(
                icon: Icon(Icons.send),
                onPressed: chat.isConnected ? _sendMessage : null,
              );
            },
          ),
        ],
      ),
    );
  }
  
  void _sendMessage() {
    final content = _controller.text.trim();
    if (content.isEmpty) return;
    
    Provider.of<ChatProvider>(context, listen: false).sendMessage(content);
    _controller.clear();
    
    // Scroll to bottom
    Future.delayed(Duration(milliseconds: 100), () {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
    });
  }
}
```

---

## Testing WebSocket

### Using `websocat` (Command Line)

```bash
# Install websocat
brew install websocat  # macOS
cargo install websocat  # via Rust

# Connect
websocat "ws://localhost:8080/ws/chat?token=YOUR_JWT_TOKEN"

# Send messages
{"content": "Hello! I'm 14 weeks pregnant"}
{"content": "When will my baby start kicking?"}
```

### Using Dart Test

```dart
import 'package:test/test.dart';

void main() {
  test('WebSocket connection', () async {
    final chatService = ChatService();
    await chatService.connect('test-token');
    
    expect(chatService.isConnected, isTrue);
    
    chatService.dispose();
  });
  
  test('Send and receive message', () async {
    final chatService = ChatService();
    await chatService.connect('test-token');
    
    final messages = <String>[];
    chatService.stream.listen((message) {
      messages.add(message);
    });
    
    chatService.sendMessage('Hello');
    
    await Future.delayed(Duration(seconds: 1));
    expect(messages.isNotEmpty, isTrue);
    
    chatService.dispose();
  });
}
```

---

## Production Considerations

### 1. Reconnection Strategy

```dart
class ReconnectionManager {
  int _retryCount = 0;
  static const int maxRetries = 5;
  
  Future<void> reconnect(ChatProvider chat, String token) async {
    if (_retryCount >= maxRetries) {
      _showMaxRetriesError();
      return;
    }
    
    _retryCount++;
    final delay = Duration(seconds: _retryCount * 2); // Exponential backoff
    
    await Future.delayed(delay);
    
    try {
      await chat.connect(token);
      _retryCount = 0; // Reset on success
    } catch (e) {
      reconnect(chat, token); // Retry
    }
  }
}
```

### 2. Network Change Detection

```dart
import 'package:connectivity_plus/connectivity_plus.dart';

class NetworkMonitor {
  final ChatProvider _chat;
  final String _token;
  
  void startMonitoring() {
    Connectivity().onConnectivityChanged.listen((result) {
      if (result != ConnectivityResult.none) {
        // Network restored, reconnect
        _chat.connect(_token);
      }
    });
  }
}
```

### 3. Message Queue (Offline Support)

```dart
class MessageQueue {
  final List<String> _pending = [];
  
  void queueMessage(String content) {
    _pending.add(content);
  }
  
  void sendPending(ChatProvider chat) {
    for (final message in _pending) {
      chat.sendMessage(message);
    }
    _pending.clear();
  }
}
```

### 4. Message Persistence

```dart
import 'package:hive/hive.dart';

class MessageRepository {
  static const String boxName = 'messages';
  
  Future<void> saveMessage(Message message) async {
    final box = await Hive.openBox<Message>(boxName);
    await box.add(message);
  }
  
  Future<List<Message>> loadMessages() async {
    final box = await Hive.openBox<Message>(boxName);
    return box.values.toList();
  }
}
```

---

## Common Issues & Solutions

### Issue: Messages not streaming
**Cause:** Not handling `type: "message"` properly
**Solution:** Concatenate chunks, don't replace

### Issue: Connection keeps dropping
**Cause:** Token expired or network issues
**Solution:** Implement reconnection with exponential backoff

### Issue: Rate limit errors
**Cause:** Sending messages too quickly
**Solution:** Track message count client-side (10/min limit)

### Issue: UI freezes during streaming
**Cause:** Updating UI on every chunk
**Solution:** Debounce UI updates (update every 100ms, not every chunk)

### Issue: Calendar suggestions not showing
**Cause:** Not handling `type: "calendar"` messages
**Solution:** Add switch case for calendar type

---

## Security Best Practices

1. **Never hardcode tokens** - Store in secure storage (flutter_secure_storage)
2. **Validate token before connecting** - Check expiry client-side
3. **Close connections properly** - Always call `dispose()` in Flutter lifecycle
4. **Sanitize user input** - Trim whitespace, validate length
5. **Don't log sensitive data** - Redact messages in debug logs

---

## Frequently Asked Questions

### Q: Which languages are supported?

**A:** Currently **English (en)**, **Spanish (es)**, and **French (fr)**.

- Fallback responses exist for EN/ES/FR
- System prompts support EN/ES/FR
- Admin can add more languages via database, but requires translation work
- All three languages have complete fallback coverage

**Frontend should:**
- Detect device language on first launch
- Default to English if device language is not EN/ES/FR
- Allow manual language switch in settings (English/Spanish/French)
- Store user's language preference for super-prompt construction

---

### Q: Should I use Provider, Riverpod, or Bloc?

**A:** The examples use Provider for simplicity, but **any state management solution works**. The key pattern is:

1. Manage WebSocket connection lifecycle
2. Listen to incoming messages
3. Concatenate streaming chunks
4. Notify UI on updates

Choose based on your team's preference. The `ChatProvider` example can be adapted to Riverpod `StateNotifier` or Bloc `Cubit` easily.

---

### Q: How do I implement audio chat (voice recording)?

**A:** The backend is **text-only**. Audio support is "indirect" - you must:

1. **Record audio** on mobile (flutter_sound, record)
2. **Convert to text** using client-side STT:
   - Google Cloud Speech-to-Text
   - AWS Transcribe
   - Device native STT (speech_to_text package)
3. **Send text** to WebSocket (same as typed messages)
4. **Receive text** response from AI
5. **Convert to speech** using TTS:
   - flutter_tts package
   - Google Cloud TTS
   - Device native TTS

**Backend does NOT provide:**
- ❌ STT endpoint
- ❌ TTS endpoint
- ❌ Audio file uploads

**Example flow:**
```dart
// 1. Record audio
final audioFile = await recorder.stop();

// 2. Send to STT service (client-side)
final text = await googleSpeech.recognize(audioFile);

// 3. Send text to backend via WebSocket
chatProvider.sendMessage(text);

// 4. Receive text response
// 5. Convert to speech (client-side)
await flutterTts.speak(response);
```

---

### Q: What about admin language management UI?

**A:** Admin features are **out of scope** for mobile app integration.

The admin endpoints (`/api/admin/languages`) are designed for a **separate web dashboard**, not the mobile app. Mobile users cannot:
- Add new languages
- Enable/disable languages
- Modify system prompts

If you're building the **admin web dashboard** (separate project), see:
- [`API.md`](API.md) - Admin endpoints (HTTP only, no WebSocket)
- [`BACKEND_SPEC.md`](BACKEND_SPEC.md) - Admin language workflow

---

### Q: How do I test WebSocket integration?

**A:** Use mocking for unit tests:

#### Mock WebSocket (Dart Test)

```dart
import 'package:mockito/mockito.dart';
import 'package:test/test.dart';

class MockWebSocketChannel extends Mock implements WebSocketChannel {}

void main() {
  test('ChatProvider handles streaming messages', () async {
    final mockChannel = MockWebSocketChannel();
    final streamController = StreamController<String>();
    
    when(mockChannel.stream).thenAnswer((_) => streamController.stream);
    
    final chatProvider = ChatProvider(channel: mockChannel);
    
    // Simulate streaming chunks
    streamController.add('{"type":"message","content":"Hello "}');
    streamController.add('{"type":"message","content":"there!"}');
    streamController.add('{"type":"done"}');
    
    await Future.delayed(Duration(milliseconds: 100));
    
    expect(chatProvider.messages.last.content, equals('Hello there!'));
    expect(chatProvider.messages.last.isStreaming, isFalse);
  });
  
  test('ChatProvider handles rate limit error', () async {
    final mockChannel = MockWebSocketChannel();
    final streamController = StreamController<String>();
    
    when(mockChannel.stream).thenAnswer((_) => streamController.stream);
    
    final chatProvider = ChatProvider(channel: mockChannel);
    
    // Simulate rate limit error
    streamController.add('{"type":"error","message":"Rate limit exceeded"}');
    
    await Future.delayed(Duration(milliseconds: 100));
    
    expect(chatProvider.hasError, isTrue);
    expect(chatProvider.errorMessage, contains('Rate limit'));
  });
}
```

#### Integration Testing (Real Backend)

```dart
// test/integration/websocket_test.dart
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  
  testWidgets('Send message and receive response', (tester) async {
    // Launch app
    await tester.pumpWidget(MyApp());
    
    // Login first
    await tester.enterText(find.byKey(Key('email')), 'test@example.com');
    await tester.enterText(find.byKey(Key('password')), 'password');
    await tester.tap(find.byKey(Key('login_button')));
    await tester.pumpAndSettle();
    
    // Navigate to chat
    await tester.tap(find.byIcon(Icons.chat));
    await tester.pumpAndSettle();
    
    // Send message
    await tester.enterText(find.byKey(Key('message_input')), 'Hello');
    await tester.tap(find.byKey(Key('send_button')));
    
    // Wait for response (streaming)
    await tester.pumpAndSettle(Duration(seconds: 5));
    
    // Verify AI response received
    expect(find.text('Hello'), findsOneWidget);
    expect(find.byType(MessageBubble), findsAtLeastNWidgets(2)); // User + AI
  });
}
```

#### Mock WebSocket Server (for Development)

```dart
// test/mocks/mock_websocket_server.dart
import 'dart:io';

class MockWebSocketServer {
  HttpServer? _server;
  
  Future<void> start() async {
    _server = await HttpServer.bind('localhost', 8080);
    
    _server!.transform(WebSocketTransformer()).listen((WebSocket ws) {
      ws.listen((message) {
        final data = jsonDecode(message);
        
        // Simulate streaming response
        ws.add('{"type":"message","content":"Mock "}');
        Future.delayed(Duration(milliseconds: 100), () {
          ws.add('{"type":"message","content":"response"}');
        });
        Future.delayed(Duration(milliseconds: 200), () {
          ws.add('{"type":"done"}');
        });
      });
    });
  }
  
  Future<void> stop() async {
    await _server?.close();
  }
}

// Usage in tests
setUp(() async {
  mockServer = MockWebSocketServer();
  await mockServer.start();
});

tearDown(() async {
  await mockServer.stop();
});
```

---

### Q: Where is the Flutter code?

**A:** This is the **backend repository** - there is no Flutter code here.

This guide helps you **integrate your Flutter app** with this backend. You will:

1. **Create a new Flutter project** (or use existing)
2. **Copy the code examples** from this guide
3. **Customize** for your app's architecture
4. **Connect** to this backend's WebSocket endpoint

**Repository structure:**
```
momlaunchpad-be/          ← You are here (Backend - Go)
  ├── cmd/server/
  ├── internal/
  └── WEBSOCKET_GUIDE.md  ← Integration docs for Flutter devs

momlaunchpad-mobile/      ← Your Flutter app (separate repo)
  ├── lib/
  │   ├── providers/
  │   │   └── chat_provider.dart    ← Copy from this guide
  │   └── screens/
  │       └── chat_screen.dart      ← Copy from this guide
  └── pubspec.yaml
```

**Next steps for frontend team:**
1. Create Flutter project: `flutter create momlaunchpad_mobile`
2. Add dependencies: `web_socket_channel`, `provider`, `flutter_secure_storage`
3. Implement `ChatProvider` (copy from this guide)
4. Build chat UI (copy `ChatScreen` example)
5. Connect to backend: `ws://your-backend-url/ws/chat`

---

## Next Steps (Frontend Implementation)

These are **frontend-only tasks** - the backend is complete and ready:

1. **Implement `ChatProvider`** - Copy the example above, customize for your state management (Provider/Riverpod/Bloc)
2. **Add reconnection logic** - Use the `ReconnectionManager` example for production resilience
3. **Message persistence** - Use Hive/SharedPreferences to cache chat history offline
4. **Network monitoring** - Add `connectivity_plus` to detect network changes and auto-reconnect
5. **Test with backend** - Backend WebSocket is fully functional at `/ws/chat`

**Backend Status:** ✅ Complete - WebSocket, rate limiting, error handling, and streaming all implemented.

For complete API documentation, see [`API.md`](API.md).
