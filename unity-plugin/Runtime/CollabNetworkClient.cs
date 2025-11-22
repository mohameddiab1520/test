using System;
using System.Text;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace Collab.Unity
{
    /// <summary>
    /// Handles HTTP and WebSocket communication with the collaboration server
    /// </summary>
    public class CollabNetworkClient : MonoBehaviour
    {
        private CollabConfig config;
        private string accessToken;
        private string userId;
        private WebSocket webSocket;

        public event Action OnConnected;
        public event Action OnDisconnected;
        public event Action<string> OnMessage;
        public event Action<string> OnError;

        public string UserId => userId;
        public string AccessToken => accessToken;
        public bool IsConnected => webSocket != null && webSocket.IsConnected;

        public void Initialize(CollabConfig cfg)
        {
            config = cfg;
        }

        /// <summary>
        /// Authenticate with the server
        /// </summary>
        public async Task<bool> Authenticate(string email, string password)
        {
            var loginData = new LoginRequest
            {
                email = email,
                password = password
            };

            string json = JsonUtility.ToJson(loginData);
            string url = $"{config.apiUrl}/api/v1/auth/login";

            using (UnityWebRequest request = new UnityWebRequest(url, "POST"))
            {
                byte[] bodyRaw = Encoding.UTF8.GetBytes(json);
                request.uploadHandler = new UploadHandlerRaw(bodyRaw);
                request.downloadHandler = new DownloadHandlerBuffer();
                request.SetRequestHeader("Content-Type", "application/json");

                var operation = request.SendWebRequest();

                while (!operation.isDone)
                {
                    await Task.Yield();
                }

                if (request.result == UnityWebRequest.Result.Success)
                {
                    var response = JsonUtility.FromJson<LoginResponse>(request.downloadHandler.text);
                    accessToken = response.accessToken;
                    userId = response.userId ?? "user-" + Guid.NewGuid().ToString();

                    Log($"Authentication successful. User ID: {userId}");
                    return true;
                }
                else
                {
                    OnError?.Invoke($"Authentication failed: {request.error}");
                    return false;
                }
            }
        }

        /// <summary>
        /// Connect to WebSocket server for a specific session
        /// </summary>
        public async Task ConnectWebSocket(string sessionId)
        {
            try
            {
                // Build WebSocket URL: ws://host/api/v1/sessions/{sessionId}/ws
                string wsUrl = config.wsUrl.Replace("http://", "ws://").Replace("https://", "wss://");
                wsUrl = $"{wsUrl}/api/v1/sessions/{sessionId}/ws";

                webSocket = new WebSocket(wsUrl, accessToken);

                webSocket.OnOpen += HandleWebSocketOpen;
                webSocket.OnClose += HandleWebSocketClose;
                webSocket.OnMessage += HandleWebSocketMessage;
                webSocket.OnError += HandleWebSocketError;

                await webSocket.Connect();
                Log($"WebSocket connecting to: {wsUrl}");
            }
            catch (Exception ex)
            {
                OnError?.Invoke($"WebSocket connection failed: {ex.Message}");
            }
        }

        /// <summary>
        /// Send presence update
        /// </summary>
        public void SendPresenceUpdate(string status, string currentScene = null, string selectedObject = null)
        {
            var message = new WebSocketMessage
            {
                type = "presence_update",
                data = new System.Collections.Generic.Dictionary<string, object>
                {
                    { "status", status },
                    { "currentScene", currentScene },
                    { "selectedObject", selectedObject }
                },
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
            };

            string json = JsonUtility.ToJson(message);
            SendMessage(json);
        }

        /// <summary>
        /// Send scene update
        /// </summary>
        public void SendSceneUpdate(object sceneData)
        {
            var message = new WebSocketMessage
            {
                type = "scene_update",
                data = new System.Collections.Generic.Dictionary<string, object>
                {
                    { "sceneData", sceneData }
                },
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
            };

            string json = JsonUtility.ToJson(message);
            SendMessage(json);
        }

        /// <summary>
        /// Send object transform update
        /// </summary>
        public void SendObjectTransform(string objectId, Vector3 position, Quaternion rotation, Vector3 scale)
        {
            var message = new WebSocketMessage
            {
                type = "object_transform",
                data = new System.Collections.Generic.Dictionary<string, object>
                {
                    { "objectId", objectId },
                    { "position", new { x = position.x, y = position.y, z = position.z } },
                    { "rotation", new { x = rotation.x, y = rotation.y, z = rotation.z, w = rotation.w } },
                    { "scale", new { x = scale.x, y = scale.y, z = scale.z } }
                },
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
            };

            string json = JsonUtility.ToJson(message);
            SendMessage(json);
        }

        /// <summary>
        /// Send chat message
        /// </summary>
        public void SendChatMessage(string message)
        {
            var wsMessage = new WebSocketMessage
            {
                type = "chat_message",
                data = new System.Collections.Generic.Dictionary<string, object>
                {
                    { "message", message }
                },
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds()
            };

            string json = JsonUtility.ToJson(wsMessage);
            SendMessage(json);
        }

        /// <summary>
        /// Create a new session
        /// </summary>
        public async Task<SessionData> CreateSession(string sessionName, string projectId)
        {
            var sessionRequest = new CreateSessionRequest
            {
                name = sessionName,
                projectId = projectId
            };

            string json = JsonUtility.ToJson(sessionRequest);
            string url = $"{config.apiUrl}/api/v1/sessions";

            using (UnityWebRequest request = new UnityWebRequest(url, "POST"))
            {
                byte[] bodyRaw = Encoding.UTF8.GetBytes(json);
                request.uploadHandler = new UploadHandlerRaw(bodyRaw);
                request.downloadHandler = new DownloadHandlerBuffer();
                request.SetRequestHeader("Content-Type", "application/json");
                request.SetRequestHeader("Authorization", $"Bearer {accessToken}");

                var operation = request.SendWebRequest();

                while (!operation.isDone)
                {
                    await Task.Yield();
                }

                if (request.result == UnityWebRequest.Result.Success)
                {
                    var response = JsonUtility.FromJson<SessionData>(request.downloadHandler.text);
                    return response;
                }
                else
                {
                    OnError?.Invoke($"Failed to create session: {request.error}");
                    return null;
                }
            }
        }

        /// <summary>
        /// Join an existing session
        /// </summary>
        public async Task<bool> JoinSession(string sessionId)
        {
            string url = $"{config.apiUrl}/api/v1/sessions/{sessionId}/join";

            using (UnityWebRequest request = new UnityWebRequest(url, "POST"))
            {
                request.downloadHandler = new DownloadHandlerBuffer();
                request.SetRequestHeader("Authorization", $"Bearer {accessToken}");

                var operation = request.SendWebRequest();

                while (!operation.isDone)
                {
                    await Task.Yield();
                }

                if (request.result == UnityWebRequest.Result.Success)
                {
                    Log($"Joined session: {sessionId}");
                    return true;
                }
                else
                {
                    OnError?.Invoke($"Failed to join session: {request.error}");
                    return false;
                }
            }
        }

        /// <summary>
        /// Leave a session
        /// </summary>
        public async Task<bool> LeaveSession(string sessionId)
        {
            string url = $"{config.apiUrl}/api/v1/sessions/{sessionId}/leave";

            using (UnityWebRequest request = new UnityWebRequest(url, "POST"))
            {
                request.downloadHandler = new DownloadHandlerBuffer();
                request.SetRequestHeader("Authorization", $"Bearer {accessToken}");

                var operation = request.SendWebRequest();

                while (!operation.isDone)
                {
                    await Task.Yield();
                }

                return request.result == UnityWebRequest.Result.Success;
            }
        }

        /// <summary>
        /// Send a message through WebSocket
        /// </summary>
        public void SendMessage(string message)
        {
            if (webSocket != null && webSocket.IsConnected)
            {
                webSocket.Send(message);
            }
            else
            {
                OnError?.Invoke("WebSocket not connected");
            }
        }

        /// <summary>
        /// Disconnect from server
        /// </summary>
        public void Disconnect()
        {
            if (webSocket != null)
            {
                webSocket.Close();
                webSocket = null;
            }

            accessToken = null;
            userId = null;
        }

        private void HandleWebSocketOpen()
        {
            Log("WebSocket connected");
            OnConnected?.Invoke();
        }

        private void HandleWebSocketClose()
        {
            Log("WebSocket disconnected");
            OnDisconnected?.Invoke();
        }

        private void HandleWebSocketMessage(string message)
        {
            Log($"WebSocket message: {message}");
            OnMessage?.Invoke(message);
        }

        private void HandleWebSocketError(string error)
        {
            Debug.LogError($"[CollabNetworkClient] WebSocket error: {error}");
            OnError?.Invoke(error);
        }

        private void Log(string message)
        {
            if (config != null && config.debugMode)
            {
                Debug.Log($"[CollabNetworkClient] {message}");
            }
        }

        private void Update()
        {
            // Process WebSocket messages on main thread
            if (webSocket != null)
            {
                webSocket.Update();
            }
        }

        private void OnDestroy()
        {
            Disconnect();
        }
    }

    // Data classes for API requests/responses
    [Serializable]
    public class LoginRequest
    {
        public string email;
        public string password;
    }

    [Serializable]
    public class LoginResponse
    {
        public string accessToken;
        public string refreshToken;
        public int expiresIn;
        public string userId;
    }

    [Serializable]
    public class CreateSessionRequest
    {
        public string name;
        public string projectId;
    }

    [Serializable]
    public class SessionData
    {
        public string id;
        public string name;
        public string projectId;
        public string ownerId;
        public string status;
    }

    [Serializable]
    public class WebSocketMessage
    {
        public string type;
        public string sessionId;
        public string userId;
        public System.Collections.Generic.Dictionary<string, object> data;
        public long timestamp;
    }

    /// <summary>
    /// Simple WebSocket implementation for Unity
    /// Note: In production, use a library like WebSocketSharp or NativeWebSocket
    /// </summary>
    public class WebSocket
    {
        private string url;
        private string token;
        public bool IsConnected { get; private set; }

        public event Action OnOpen;
        public event Action OnClose;
        public event Action<string> OnMessage;
        public event Action<string> OnError;

        public WebSocket(string url, string authToken = null)
        {
            this.url = url;
            this.token = authToken;
        }

        public async Task Connect()
        {
            // Placeholder for WebSocket connection
            // In a real implementation, use WebSocketSharp or browser WebSocket API
            await Task.Delay(100);
            IsConnected = true;
            OnOpen?.Invoke();
        }

        public void Send(string message)
        {
            if (!IsConnected)
            {
                OnError?.Invoke("Not connected");
                return;
            }

            // Placeholder for sending message
            Debug.Log($"[WebSocket] Sending: {message}");
        }

        public void Close()
        {
            if (!IsConnected) return;

            IsConnected = false;
            OnClose?.Invoke();
        }

        public void Update()
        {
            // Placeholder for processing incoming messages
            // In a real implementation, this would dispatch queued messages
        }
    }
}
