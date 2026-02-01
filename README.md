# DragonFox MediaSync Server

WebSocket relay server for synchronizing media playback between clients.

## How It Works

Clients connect to a room. When one client sends a play/pause/seek event, all other clients in the same room receive it.

## Running

```bash
go run .
```

Environment variables:
- `PORT` — server port (default: 8080)
- `LOG_LEVEL` — debug, info, warn, error (default: info)

## Client Integration

### Connect

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?room=my-room');
```

Room parameter is optional. Default room is `default`.

### Message Format

```typescript
interface Message {
  type: string;           // "play" | "pause" | "toggle" | "seek" | "ping"
  position?: number;      // milliseconds (for seek)
  timestamp: number;      // unix ms (for latency calculation)
  clientId?: string;      // sender ID (added by server)
}
```

### Send Events

```javascript
// Play
ws.send(JSON.stringify({ type: 'play', timestamp: Date.now() }));

// Pause
ws.send(JSON.stringify({ type: 'pause', timestamp: Date.now() }));

// Toggle play/pause
ws.send(JSON.stringify({ type: 'toggle', timestamp: Date.now() }));

// Seek to position
ws.send(JSON.stringify({ type: 'seek', position: 30000, timestamp: Date.now() }));

// Ping (server responds with pong)
ws.send(JSON.stringify({ type: 'ping', timestamp: Date.now() }));
```

### Receive Events

```javascript
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  switch (msg.type) {
    case 'play':
      player.play();
      break;
    case 'pause':
      player.pause();
      break;
    case 'toggle':
      player.paused ? player.play() : player.pause();
      break;
    case 'seek':
      player.currentTime = msg.position / 1000;
      break;
    case 'pong':
      const latency = Date.now() - msg.timestamp;
      console.log(`Latency: ${latency}ms`);
      break;
  }
};
```

## HTTP Endpoints

- `GET /health` — `{"status": "ok"}`
- `GET /stats` — `{"rooms": 5, "clients": 12}`
