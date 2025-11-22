# Unity Collaboration Platform - Unity Plugin Architecture

## Table of Contents

1. [Plugin Overview](#plugin-overview)
2. [Architecture](#architecture)
3. [Core Components](#core-components)
4. [Networking](#networking)
5. [Scene Synchronization](#scene-synchronization)
6. [Asset Management](#asset-management)
7. [UI/UX](#uiux)
8. [Installation & Setup](#installation--setup)

---

## Plugin Overview

### Features

- **Real-time Scene Collaboration**: Multiple developers editing the same scene
- **Asset Synchronization**: Automatic sync of project assets
- **Voice Chat**: Built-in voice communication
- **Presence Indicators**: See where teammates are working
- **Conflict Resolution**: Smart handling of concurrent edits
- **Version Control Integration**: Works with Git/Perforce
- **Offline Support**: Local caching and sync when online

### Requirements

- Unity 2022.3 LTS or higher
- .NET Standard 2.1
- Windows, macOS, or Linux Editor

---

## Architecture

### High-Level Structure

```
UnityCollabPlugin/
├── Runtime/                     # Runtime scripts
│   ├── Core/
│   │   ├── CollaborationManager.cs
│   │   ├── NetworkManager.cs
│   │   └── SessionManager.cs
│   ├── Sync/
│   │   ├── SceneSynchronizer.cs
│   │   ├── ObjectTracker.cs
│   │   └── OperationQueue.cs
│   ├── Networking/
│   │   ├── WebSocketClient.cs
│   │   ├── HttpClient.cs
│   │   └── MessageSerializer.cs
│   ├── Assets/
│   │   ├── AssetUploader.cs
│   │   ├── AssetDownloader.cs
│   │   └── AssetCache.cs
│   ├── Voice/
│   │   ├── VoiceManager.cs
│   │   └── AudioProcessor.cs
│   └── UI/
│       ├── CollaborationPanel.cs
│       ├── ParticipantList.cs
│       └── PresenceIndicator.cs
├── Editor/                      # Editor scripts
│   ├── CollaborationWindow.cs
│   ├── SessionCreator.cs
│   ├── SettingsProvider.cs
│   └── SceneValidator.cs
├── Resources/
│   ├── UI/
│   │   └── CollabUI.uxml
│   └── Icons/
├── Tests/
│   ├── Runtime/
│   └── Editor/
└── package.json
```

---

## Core Components

### 1. CollaborationManager

**Purpose**: Central coordinator for all collaboration features

```csharp
using System;
using System.Threading.Tasks;
using UnityEngine;

namespace UnityCollab.Core
{
    /// <summary>
    /// Main entry point for Unity Collaboration Plugin
    /// </summary>
    public class CollaborationManager : MonoBehaviour
    {
        private static CollaborationManager _instance;
        public static CollaborationManager Instance
        {
            get
            {
                if (_instance == null)
                {
                    var go = new GameObject("CollaborationManager");
                    _instance = go.AddComponent<CollaborationManager>();
                    DontDestroyOnLoad(go);
                }
                return _instance;
            }
        }

        // Components
        private NetworkManager _networkManager;
        private SessionManager _sessionManager;
        private SceneSynchronizer _sceneSynchronizer;
        private AssetManager _assetManager;
        private VoiceManager _voiceManager;

        // Events
        public event Action<Session> OnSessionJoined;
        public event Action OnSessionLeft;
        public event Action<Participant> OnParticipantJoined;
        public event Action<Participant> OnParticipantLeft;
        public event Action<SyncOperation> OnOperationReceived;

        // State
        public bool IsConnected { get; private set; }
        public Session CurrentSession { get; private set; }
        public User CurrentUser { get; private set; }

        private void Awake()
        {
            if (_instance != null && _instance != this)
            {
                Destroy(gameObject);
                return;
            }

            _instance = this;
            DontDestroyOnLoad(gameObject);

            InitializeComponents();
        }

        private void InitializeComponents()
        {
            _networkManager = new NetworkManager();
            _sessionManager = new SessionManager(_networkManager);
            _sceneSynchronizer = new SceneSynchronizer(_networkManager);
            _assetManager = new AssetManager(_networkManager);
            _voiceManager = new VoiceManager(_networkManager);

            SubscribeToEvents();
        }

        private void SubscribeToEvents()
        {
            _networkManager.OnConnected += HandleConnected;
            _networkManager.OnDisconnected += HandleDisconnected;
            _networkManager.OnMessageReceived += HandleMessage;

            _sessionManager.OnSessionJoined += HandleSessionJoined;
            _sessionManager.OnSessionLeft += HandleSessionLeft;
        }

        /// <summary>
        /// Authenticate user and connect to server
        /// </summary>
        public async Task<bool> ConnectAsync(string email, string password)
        {
            try
            {
                // Authenticate
                var authResult = await _networkManager.AuthenticateAsync(email, password);
                if (!authResult.Success)
                {
                    Debug.LogError($"Authentication failed: {authResult.Error}");
                    return false;
                }

                CurrentUser = authResult.User;
                IsConnected = true;

                Debug.Log($"Connected as {CurrentUser.Name}");
                return true;
            }
            catch (Exception ex)
            {
                Debug.LogError($"Connection failed: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Create a new collaboration session
        /// </summary>
        public async Task<Session> CreateSessionAsync(string sessionName, SessionSettings settings = null)
        {
            if (!IsConnected)
            {
                throw new InvalidOperationException("Not connected. Call ConnectAsync first.");
            }

            var session = await _sessionManager.CreateSessionAsync(sessionName, settings);
            return session;
        }

        /// <summary>
        /// Join an existing collaboration session
        /// </summary>
        public async Task<bool> JoinSessionAsync(string sessionId)
        {
            if (!IsConnected)
            {
                throw new InvalidOperationException("Not connected. Call ConnectAsync first.");
            }

            var success = await _sessionManager.JoinSessionAsync(sessionId);
            return success;
        }

        /// <summary>
        /// Leave current session
        /// </summary>
        public async Task LeaveSessionAsync()
        {
            if (CurrentSession == null)
            {
                Debug.LogWarning("Not in a session");
                return;
            }

            await _sessionManager.LeaveSessionAsync();
        }

        private void HandleConnected()
        {
            Debug.Log("Connected to collaboration server");
        }

        private void HandleDisconnected()
        {
            Debug.Log("Disconnected from collaboration server");
            IsConnected = false;
            CurrentSession = null;
        }

        private void HandleSessionJoined(Session session)
        {
            CurrentSession = session;
            _sceneSynchronizer.StartSynchronization(session.SessionId);
            _voiceManager.JoinVoiceChannel(session.SessionId);

            OnSessionJoined?.Invoke(session);
            Debug.Log($"Joined session: {session.Name}");
        }

        private void HandleSessionLeft()
        {
            _sceneSynchronizer.StopSynchronization();
            _voiceManager.LeaveVoiceChannel();

            CurrentSession = null;
            OnSessionLeft?.Invoke();
            Debug.Log("Left session");
        }

        private void HandleMessage(NetworkMessage message)
        {
            // Route messages to appropriate handlers
            switch (message.Type)
            {
                case MessageType.Sync:
                    _sceneSynchronizer.HandleSyncMessage(message);
                    break;
                case MessageType.ParticipantJoined:
                    OnParticipantJoined?.Invoke(message.GetData<Participant>());
                    break;
                case MessageType.ParticipantLeft:
                    OnParticipantLeft?.Invoke(message.GetData<Participant>());
                    break;
            }
        }

        private void Update()
        {
            // Process queued operations on main thread
            _sceneSynchronizer?.ProcessQueue();
        }

        private void OnApplicationQuit()
        {
            LeaveSessionAsync().Wait();
            _networkManager?.Disconnect();
        }
    }
}
```

### 2. NetworkManager

**Purpose**: Handle all network communication (HTTP and WebSocket)

```csharp
using System;
using System.Threading.Tasks;
using UnityEngine;
using WebSocketSharp;

namespace UnityCollab.Networking
{
    public class NetworkManager
    {
        private const string API_BASE_URL = "https://api.collab.example.com/v1";
        private const string WS_URL = "wss://sync.collab.example.com/ws";

        private HttpClient _httpClient;
        private WebSocket _webSocket;
        private string _accessToken;

        // Events
        public event Action OnConnected;
        public event Action OnDisconnected;
        public event Action<NetworkMessage> OnMessageReceived;

        public bool IsConnected => _webSocket?.IsAlive ?? false;

        public NetworkManager()
        {
            _httpClient = new HttpClient(API_BASE_URL);
        }

        /// <summary>
        /// Authenticate with server
        /// </summary>
        public async Task<AuthResult> AuthenticateAsync(string email, string password)
        {
            var response = await _httpClient.PostAsync("/auth/login", new
            {
                email = email,
                password = password
            });

            if (!response.Success)
            {
                return new AuthResult { Success = false, Error = response.Error };
            }

            _accessToken = response.Data["accessToken"].ToString();
            _httpClient.SetAuthToken(_accessToken);

            var user = JsonUtility.FromJson<User>(response.Data["user"].ToString());

            return new AuthResult
            {
                Success = true,
                User = user,
                AccessToken = _accessToken
            };
        }

        /// <summary>
        /// Connect WebSocket for real-time sync
        /// </summary>
        public async Task<bool> ConnectWebSocketAsync(string sessionId)
        {
            try
            {
                var wsUrl = $"{WS_URL}?token={_accessToken}&sessionId={sessionId}";

                _webSocket = new WebSocket(wsUrl);

                _webSocket.OnOpen += (sender, e) =>
                {
                    Debug.Log("WebSocket connected");
                    OnConnected?.Invoke();
                };

                _webSocket.OnMessage += (sender, e) =>
                {
                    var message = JsonUtility.FromJson<NetworkMessage>(e.Data);
                    OnMessageReceived?.Invoke(message);
                };

                _webSocket.OnError += (sender, e) =>
                {
                    Debug.LogError($"WebSocket error: {e.Message}");
                };

                _webSocket.OnClose += (sender, e) =>
                {
                    Debug.Log($"WebSocket closed: {e.Reason}");
                    OnDisconnected?.Invoke();
                };

                _webSocket.Connect();

                // Wait for connection
                int retries = 0;
                while (!_webSocket.IsAlive && retries < 50)
                {
                    await Task.Delay(100);
                    retries++;
                }

                return _webSocket.IsAlive;
            }
            catch (Exception ex)
            {
                Debug.LogError($"WebSocket connection failed: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Send operation over WebSocket
        /// </summary>
        public void SendOperation(SyncOperation operation)
        {
            if (!IsConnected)
            {
                Debug.LogWarning("Cannot send operation: not connected");
                return;
            }

            var message = new NetworkMessage
            {
                Type = MessageType.Operation,
                Data = JsonUtility.ToJson(operation)
            };

            _webSocket.Send(JsonUtility.ToJson(message));
        }

        /// <summary>
        /// Make HTTP request
        /// </summary>
        public async Task<ApiResponse> RequestAsync(string method, string endpoint, object data = null)
        {
            switch (method.ToUpper())
            {
                case "GET":
                    return await _httpClient.GetAsync(endpoint);
                case "POST":
                    return await _httpClient.PostAsync(endpoint, data);
                case "PUT":
                    return await _httpClient.PutAsync(endpoint, data);
                case "DELETE":
                    return await _httpClient.DeleteAsync(endpoint);
                default:
                    throw new ArgumentException($"Unsupported HTTP method: {method}");
            }
        }

        /// <summary>
        /// Disconnect
        /// </summary>
        public void Disconnect()
        {
            _webSocket?.Close();
            _webSocket = null;
        }
    }
}
```

### 3. SceneSynchronizer

**Purpose**: Track and synchronize scene changes

```csharp
using System;
using System.Collections.Generic;
using System.Linq;
using UnityEngine;
using UnityEngine.SceneManagement;

namespace UnityCollab.Sync
{
    public class SceneSynchronizer
    {
        private NetworkManager _networkManager;
        private ObjectTracker _objectTracker;
        private OperationQueue _operationQueue;

        private string _currentSessionId;
        private bool _isSyncing;
        private Dictionary<string, GameObject> _trackedObjects;

        public SceneSynchronizer(NetworkManager networkManager)
        {
            _networkManager = networkManager;
            _objectTracker = new ObjectTracker();
            _operationQueue = new OperationQueue();
            _trackedObjects = new Dictionary<string, GameObject>();
        }

        public void StartSynchronization(string sessionId)
        {
            _currentSessionId = sessionId;
            _isSyncing = true;

            // Track all objects in current scene
            TrackSceneObjects();

            // Subscribe to scene changes
            SceneManager.sceneLoaded += OnSceneLoaded;

            Debug.Log("Scene synchronization started");
        }

        public void StopSynchronization()
        {
            _isSyncing = false;
            _trackedObjects.Clear();

            SceneManager.sceneLoaded -= OnSceneLoaded;

            Debug.Log("Scene synchronization stopped");
        }

        private void TrackSceneObjects()
        {
            var scene = SceneManager.GetActiveScene();
            var rootObjects = scene.GetRootGameObjects();

            foreach (var root in rootObjects)
            {
                TrackGameObject(root, recursive: true);
            }

            Debug.Log($"Tracking {_trackedObjects.Count} objects");
        }

        private void TrackGameObject(GameObject obj, bool recursive = false)
        {
            if (obj == null) return;

            var trackable = obj.GetComponent<CollabTrackable>();
            if (trackable == null)
            {
                trackable = obj.AddComponent<CollabTrackable>();
            }

            string objectId = trackable.ObjectId;
            _trackedObjects[objectId] = obj;

            // Subscribe to changes
            trackable.OnTransformChanged += (id, transform) =>
            {
                SendTransformUpdate(id, transform);
            };

            trackable.OnComponentChanged += (id, component, property, value) =>
            {
                SendComponentUpdate(id, component, property, value);
            };

            if (recursive)
            {
                for (int i = 0; i < obj.transform.childCount; i++)
                {
                    TrackGameObject(obj.transform.GetChild(i).gameObject, recursive: true);
                }
            }
        }

        private void SendTransformUpdate(string objectId, Transform transform)
        {
            if (!_isSyncing) return;

            var operation = new SyncOperation
            {
                Type = OperationType.Update,
                ObjectId = objectId,
                Path = "transform",
                Value = new
                {
                    position = new { x = transform.position.x, y = transform.position.y, z = transform.position.z },
                    rotation = new { x = transform.rotation.x, y = transform.rotation.y, z = transform.rotation.z, w = transform.rotation.w },
                    scale = new { x = transform.localScale.x, y = transform.localScale.y, z = transform.localScale.z }
                },
                Timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
            };

            _networkManager.SendOperation(operation);
        }

        private void SendComponentUpdate(string objectId, string componentType, string property, object value)
        {
            if (!_isSyncing) return;

            var operation = new SyncOperation
            {
                Type = OperationType.Update,
                ObjectId = objectId,
                Path = $"{componentType}.{property}",
                Value = value,
                Timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
            };

            _networkManager.SendOperation(operation);
        }

        public void HandleSyncMessage(NetworkMessage message)
        {
            var operation = JsonUtility.FromJson<SyncOperation>(message.Data);

            // Queue operation for processing on main thread
            _operationQueue.Enqueue(operation);
        }

        public void ProcessQueue()
        {
            if (!_isSyncing) return;

            while (_operationQueue.TryDequeue(out var operation))
            {
                ApplyOperation(operation);
            }
        }

        private void ApplyOperation(SyncOperation operation)
        {
            if (!_trackedObjects.TryGetValue(operation.ObjectId, out var obj))
            {
                Debug.LogWarning($"Object not found: {operation.ObjectId}");
                return;
            }

            switch (operation.Type)
            {
                case OperationType.Create:
                    // Create new object
                    break;

                case OperationType.Update:
                    ApplyUpdate(obj, operation);
                    break;

                case OperationType.Delete:
                    UnityEngine.Object.Destroy(obj);
                    _trackedObjects.Remove(operation.ObjectId);
                    break;
            }
        }

        private void ApplyUpdate(GameObject obj, SyncOperation operation)
        {
            if (operation.Path == "transform")
            {
                var data = JsonUtility.FromJson<TransformData>(JsonUtility.ToJson(operation.Value));

                obj.transform.position = new Vector3(data.position.x, data.position.y, data.position.z);
                obj.transform.rotation = new Quaternion(data.rotation.x, data.rotation.y, data.rotation.z, data.rotation.w);
                obj.transform.localScale = new Vector3(data.scale.x, data.scale.y, data.scale.z);
            }
            else
            {
                // Update component property
                var parts = operation.Path.Split('.');
                var componentType = parts[0];
                var property = parts[1];

                var component = obj.GetComponent(componentType);
                if (component != null)
                {
                    var field = component.GetType().GetField(property);
                    if (field != null)
                    {
                        field.SetValue(component, operation.Value);
                    }
                }
            }
        }

        private void OnSceneLoaded(Scene scene, LoadSceneMode mode)
        {
            if (_isSyncing)
            {
                TrackSceneObjects();
            }
        }
    }

    [Serializable]
    public class TransformData
    {
        public Vector3Data position;
        public QuaternionData rotation;
        public Vector3Data scale;
    }

    [Serializable]
    public class Vector3Data
    {
        public float x, y, z;
    }

    [Serializable]
    public class QuaternionData
    {
        public float x, y, z, w;
    }
}
```

### 4. CollabTrackable Component

**Purpose**: Mark GameObjects for synchronization

```csharp
using System;
using UnityEngine;

namespace UnityCollab.Sync
{
    /// <summary>
    /// Component that marks a GameObject for collaboration tracking
    /// </summary>
    [AddComponentMenu("Collaboration/Trackable")]
    public class CollabTrackable : MonoBehaviour
    {
        [SerializeField]
        private string _objectId;

        public string ObjectId
        {
            get
            {
                if (string.IsNullOrEmpty(_objectId))
                {
                    _objectId = Guid.NewGuid().ToString();
                }
                return _objectId;
            }
        }

        // Events
        public event Action<string, Transform> OnTransformChanged;
        public event Action<string, string, string, object> OnComponentChanged;

        // Tracking
        private Vector3 _lastPosition;
        private Quaternion _lastRotation;
        private Vector3 _lastScale;

        private void Start()
        {
            _lastPosition = transform.position;
            _lastRotation = transform.rotation;
            _lastScale = transform.localScale;
        }

        private void Update()
        {
            CheckTransformChanges();
        }

        private void CheckTransformChanges()
        {
            bool changed = false;

            if (_lastPosition != transform.position)
            {
                _lastPosition = transform.position;
                changed = true;
            }

            if (_lastRotation != transform.rotation)
            {
                _lastRotation = transform.rotation;
                changed = true;
            }

            if (_lastScale != transform.localScale)
            {
                _lastScale = transform.localScale;
                changed = true;
            }

            if (changed)
            {
                OnTransformChanged?.Invoke(ObjectId, transform);
            }
        }

        /// <summary>
        /// Notify that a component property has changed
        /// </summary>
        public void NotifyComponentChanged(string componentType, string property, object value)
        {
            OnComponentChanged?.Invoke(ObjectId, componentType, property, value);
        }
    }
}
```

---

## UI/UX

### Collaboration Panel (UIToolkit)

**CollabPanel.uxml**
```xml
<ui:UXML xmlns:ui="UnityEngine.UIElements">
    <ui:VisualElement class="collab-panel">
        <!-- Header -->
        <ui:VisualElement class="header">
            <ui:Label text="Collaboration" class="title"/>
            <ui:Button name="settingsBtn" class="icon-btn">
                <ui:VisualElement class="icon settings-icon"/>
            </ui:Button>
        </ui:VisualElement>

        <!-- Session Info -->
        <ui:VisualElement name="sessionInfo" class="session-info">
            <ui:Label name="sessionName" class="session-name"/>
            <ui:Label name="participantCount" class="participant-count"/>
        </ui:VisualElement>

        <!-- Participant List -->
        <ui:ScrollView name="participantList" class="participant-list">
            <!-- Populated dynamically -->
        </ui:ScrollView>

        <!-- Actions -->
        <ui:VisualElement class="actions">
            <ui:Button name="joinBtn" text="Join Session" class="btn primary"/>
            <ui:Button name="leaveBtn" text="Leave Session" class="btn secondary"/>
        </ui:VisualElement>

        <!-- Voice Controls -->
        <ui:VisualElement class="voice-controls">
            <ui:Toggle name="muteToggle" label="Mute" class="toggle"/>
            <ui:Toggle name="deafenToggle" label="Deafen" class="toggle"/>
        </ui:VisualElement>
    </ui:VisualElement>
</ui:UXML>
```

---

## Installation & Setup

### via Unity Package Manager

1. Open Unity Package Manager
2. Click "+" → "Add package from git URL"
3. Enter: `https://github.com/yourorg/unity-collab-plugin.git`
4. Click "Add"

### Manual Installation

1. Download latest release from GitHub
2. Extract to `Packages/com.yourorg.unity-collab/`
3. Unity will automatically import the package

### Configuration

1. Open **Edit → Project Settings → Collaboration**
2. Enter API URL: `https://api.collab.example.com`
3. Click "Login" and enter credentials
4. Ready to collaborate!

---

This Unity plugin architecture provides a solid foundation for real-time collaboration within the Unity Editor with clean separation of concerns and extensible design.
