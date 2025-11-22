# Unity Collaboration Plugin

Real-time collaboration plugin for Unity Editor - Similar to Coplay Premium.

## Features

- **Real-time Scene Synchronization**: Sync GameObjects across multiple Unity editors
- **Transform Synchronization**: Position, rotation, and scale sync
- **Presence Indicators**: See who's editing what in real-time
- **Session Management**: Create and join collaboration sessions
- **Network Optimization**: Delta synchronization with configurable update rates
- **Smooth Interpolation**: Smooth remote object movements

## Installation

### Via Unity Package Manager

1. Open Unity Package Manager (Window > Package Manager)
2. Click "+" button → "Add package from git URL"
3. Enter: `https://github.com/yourorg/unity-collab-plugin.git`
4. Click "Add"

### Manual Installation

1. Download the package
2. Extract to your project's `Packages` folder
3. Unity will automatically detect and import the package

## Quick Start

### 1. Create Configuration

1. Right-click in Project window
2. Create > Collaboration > Config
3. Configure server URLs:
   - API URL: `http://localhost:3000`
   - WebSocket URL: `ws://localhost:8081`

### 2. Open Collaboration Window

1. Window > Collaboration > Collaboration Manager
2. Assign the config you created
3. Enter email and password
4. Click "Connect"

### 3. Create or Join Session

**Create Session:**
1. Enter session name
2. Enter project ID (optional)
3. Click "Create Session"

**Join Session:**
1. Enter session ID
2. Click "Join Session"

### 4. Sync GameObjects

1. Select any GameObject in your scene
2. Add Component > Collaboration > Synced Object
3. Configure what to sync (position, rotation, scale)
4. The object will now sync across all clients!

## Components

### CollabManager

Main singleton manager for collaboration features.

```csharp
// Access the manager
CollabManager.Instance.Connect();
CollabManager.Instance.CreateSession("My Session", "project-123");

// Subscribe to events
CollabManager.Instance.OnConnected += () => Debug.Log("Connected!");
CollabManager.Instance.OnSessionJoined += (sessionId) => Debug.Log($"Joined: {sessionId}");
```

### CollabSyncedObject

Component to sync a GameObject across clients.

```csharp
// Add to a GameObject
var syncedObj = gameObject.AddComponent<CollabSyncedObject>();

// Configure
syncedObj.syncPosition = true;
syncedObj.syncRotation = true;
syncedObj.useInterpolation = true;

// Transfer ownership
syncedObj.SetOwnership(true);
```

### CollabConfig

ScriptableObject for configuration.

- **apiUrl**: Backend API URL
- **wsUrl**: WebSocket server URL
- **syncInterval**: Update rate (default: 50ms)
- **enablePresence**: Show user presence
- **debugMode**: Enable debug logging

## API Reference

### CollabManager

| Method | Description |
|--------|-------------|
| `Connect()` | Connect to collaboration server |
| `Disconnect()` | Disconnect from server |
| `CreateSession(name, projectId)` | Create new session |
| `JoinSession(sessionId)` | Join existing session |
| `LeaveSession()` | Leave current session |

### Events

| Event | Description |
|-------|-------------|
| `OnConnected` | Fired when connected to server |
| `OnDisconnected` | Fired when disconnected |
| `OnSessionJoined` | Fired when session joined |
| `OnError` | Fired on error |

## Configuration

### Sync Settings

```csharp
config.syncInterval = 0.05f;  // 20 updates/sec
config.enablePresence = true;
config.enableVoiceChat = false;
```

### Network Settings

```csharp
config.apiUrl = "https://api.collab.example.com";
config.wsUrl = "wss://sync.collab.example.com";
```

## Best Practices

1. **Ownership**: Only local owner should modify synced objects
2. **Sync Rate**: Use 0.05s (20Hz) for most objects, 0.1s (10Hz) for slow objects
3. **Interpolation**: Enable for smooth remote movements
4. **Unique IDs**: Ensure object IDs are unique across scene
5. **Network Optimization**: Don't sync objects that don't change

## Troubleshooting

### Connection Issues

- Check server is running and accessible
- Verify URLs in config
- Check firewall settings
- Enable debug mode for detailed logs

### Sync Not Working

- Ensure CollabSyncedObject is attached
- Verify object has unique ID
- Check ownership settings
- Confirm you're in a session

### Performance Issues

- Reduce sync rate (increase interval)
- Disable interpolation if not needed
- Limit number of synced objects
- Use sync only for objects that change

## Examples

### Example 1: Basic Cube Sync

```csharp
// Create a cube
var cube = GameObject.CreatePrimitive(PrimitiveType.Cube);

// Add sync component
var sync = cube.AddComponent<CollabSyncedObject>();
sync.syncPosition = true;
sync.syncRotation = true;

// Move it (will sync to other clients)
cube.transform.position = new Vector3(1, 2, 3);
```

### Example 2: Player Sync

```csharp
public class Player : MonoBehaviour
{
    private CollabSyncedObject syncObj;

    void Start()
    {
        syncObj = GetComponent<CollabSyncedObject>();

        // Set ownership for local player
        if (isLocalPlayer)
        {
            syncObj.SetOwnership(true);
        }
    }

    void Update()
    {
        // Only local player can move
        if (syncObj.IsLocalOwner)
        {
            // Movement code here
            transform.Translate(Input.GetAxis("Horizontal"), 0, Input.GetAxis("Vertical"));
        }
    }
}
```

## Requirements

- Unity 2022.3 LTS or higher
- .NET Standard 2.1
- Active internet connection
- Collaboration backend server

## Support

- **Documentation**: [Full Docs](https://docs.collab.example.com)
- **Issues**: [GitHub Issues](https://github.com/yourorg/unity-collab-plugin/issues)
- **Discord**: [Join Community](https://discord.gg/example)
- **Email**: support@collab.example.com

## License

MIT License - See LICENSE file for details

## Credits

Inspired by:
- Coplay Premium
- Unity Netcode
- Mirror Networking
- Photon Unity Networking

---

**Made with ❤️ for Unity developers**
