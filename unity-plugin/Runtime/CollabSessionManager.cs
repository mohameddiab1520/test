using System;
using System.Collections.Generic;
using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Manages the current collaboration session
    /// </summary>
    public class CollabSessionManager : MonoBehaviour
    {
        private CollabNetworkClient networkClient;
        private string currentSessionId;
        private SessionData currentSession;
        private Dictionary<string, ParticipantData> participants = new Dictionary<string, ParticipantData>();

        public event Action<SessionData> OnSessionJoined;
        public event Action OnSessionLeft;
        public event Action<ParticipantData> OnParticipantJoined;
        public event Action<string> OnParticipantLeft;
        public event Action<string, object> OnSceneUpdate;
        public event Action<string, TransformData> OnObjectTransformUpdate;

        public string CurrentSessionId => currentSessionId;
        public SessionData CurrentSession => currentSession;
        public bool IsInSession => currentSession != null;

        public void Initialize(CollabNetworkClient client)
        {
            networkClient = client;
            networkClient.OnMessage += HandleMessage;
        }

        /// <summary>
        /// Create and join a new session
        /// </summary>
        public async void CreateAndJoinSession(string sessionName, string projectId)
        {
            if (networkClient == null || string.IsNullOrEmpty(networkClient.AccessToken))
            {
                Debug.LogError("[CollabSessionManager] Not authenticated");
                return;
            }

            try
            {
                // Create session
                currentSession = await networkClient.CreateSession(sessionName, projectId);
                if (currentSession == null)
                {
                    Debug.LogError("[CollabSessionManager] Failed to create session");
                    return;
                }

                currentSessionId = currentSession.id;
                Debug.Log($"[CollabSessionManager] Session created: {currentSessionId}");

                // Connect WebSocket
                await networkClient.ConnectWebSocket(currentSessionId);

                // Notify listeners
                OnSessionJoined?.Invoke(currentSession);

                Debug.Log($"[CollabSessionManager] Joined session: {currentSession.name}");
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabSessionManager] Error creating session: {ex.Message}");
            }
        }

        /// <summary>
        /// Join an existing session
        /// </summary>
        public async void JoinSession(string sessionId)
        {
            if (networkClient == null || string.IsNullOrEmpty(networkClient.AccessToken))
            {
                Debug.LogError("[CollabSessionManager] Not authenticated");
                return;
            }

            try
            {
                // Join session
                bool success = await networkClient.JoinSession(sessionId);
                if (!success)
                {
                    Debug.LogError("[CollabSessionManager] Failed to join session");
                    return;
                }

                currentSessionId = sessionId;

                // Connect WebSocket
                await networkClient.ConnectWebSocket(sessionId);

                Debug.Log($"[CollabSessionManager] Joined session: {sessionId}");
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabSessionManager] Error joining session: {ex.Message}");
            }
        }

        /// <summary>
        /// Leave the current session
        /// </summary>
        public async void LeaveSession()
        {
            if (string.IsNullOrEmpty(currentSessionId))
            {
                Debug.LogWarning("[CollabSessionManager] No active session");
                return;
            }

            try
            {
                await networkClient.LeaveSession(currentSessionId);
                networkClient.Disconnect();

                currentSessionId = null;
                currentSession = null;
                participants.Clear();

                OnSessionLeft?.Invoke();

                Debug.Log("[CollabSessionManager] Left session");
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabSessionManager] Error leaving session: {ex.Message}");
            }
        }

        /// <summary>
        /// Update presence
        /// </summary>
        public void UpdatePresence(string status, string currentScene = null, string selectedObject = null)
        {
            if (!IsInSession)
            {
                Debug.LogWarning("[CollabSessionManager] No active session");
                return;
            }

            networkClient.SendPresenceUpdate(status, currentScene, selectedObject);
        }

        /// <summary>
        /// Broadcast object transform
        /// </summary>
        public void BroadcastTransform(string objectId, Vector3 position, Quaternion rotation, Vector3 scale)
        {
            if (!IsInSession)
            {
                Debug.LogWarning("[CollabSessionManager] No active session");
                return;
            }

            networkClient.SendObjectTransform(objectId, position, rotation, scale);
        }

        /// <summary>
        /// Send chat message
        /// </summary>
        public void SendChatMessage(string message)
        {
            if (!IsInSession)
            {
                Debug.LogWarning("[CollabSessionManager] No active session");
                return;
            }

            networkClient.SendChatMessage(message);
        }

        /// <summary>
        /// Handle incoming WebSocket messages
        /// </summary>
        private void HandleMessage(string jsonMessage)
        {
            try
            {
                var message = JsonUtility.FromJson<WebSocketMessage>(jsonMessage);

                switch (message.type)
                {
                    case "participant_joined":
                        HandleParticipantJoined(message);
                        break;

                    case "participant_left":
                        HandleParticipantLeft(message);
                        break;

                    case "presence_update":
                        HandlePresenceUpdate(message);
                        break;

                    case "scene_update":
                        HandleSceneUpdate(message);
                        break;

                    case "object_transform":
                        HandleObjectTransform(message);
                        break;

                    case "chat_message":
                        HandleChatMessage(message);
                        break;

                    default:
                        Debug.LogWarning($"[CollabSessionManager] Unknown message type: {message.type}");
                        break;
                }
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabSessionManager] Error handling message: {ex.Message}");
            }
        }

        private void HandleParticipantJoined(WebSocketMessage message)
        {
            var participant = new ParticipantData
            {
                userId = message.userId,
                status = "online"
            };

            participants[message.userId] = participant;
            OnParticipantJoined?.Invoke(participant);

            Debug.Log($"[CollabSessionManager] Participant joined: {message.userId}");
        }

        private void HandleParticipantLeft(WebSocketMessage message)
        {
            participants.Remove(message.userId);
            OnParticipantLeft?.Invoke(message.userId);

            Debug.Log($"[CollabSessionManager] Participant left: {message.userId}");
        }

        private void HandlePresenceUpdate(WebSocketMessage message)
        {
            if (participants.ContainsKey(message.userId))
            {
                // Update participant presence
                Debug.Log($"[CollabSessionManager] Presence updated for: {message.userId}");
            }
        }

        private void HandleSceneUpdate(WebSocketMessage message)
        {
            OnSceneUpdate?.Invoke(message.userId, message.data);
            Debug.Log($"[CollabSessionManager] Scene updated by: {message.userId}");
        }

        private void HandleObjectTransform(WebSocketMessage message)
        {
            if (message.data != null && message.data.ContainsKey("objectId"))
            {
                // Parse transform data
                var transformData = new TransformData();
                // TODO: Parse position, rotation, scale from message.data

                OnObjectTransformUpdate?.Invoke(message.data["objectId"].ToString(), transformData);
            }
        }

        private void HandleChatMessage(WebSocketMessage message)
        {
            Debug.Log($"[CollabSessionManager] Chat from {message.userId}: {message.data["message"]}");
        }

        private void OnDestroy()
        {
            if (networkClient != null)
            {
                networkClient.OnMessage -= HandleMessage;
            }
        }
    }

    [Serializable]
    public class ParticipantData
    {
        public string userId;
        public string username;
        public string avatarUrl;
        public string status;
        public string currentScene;
        public string selectedObject;
    }

    [Serializable]
    public class TransformData
    {
        public Vector3 position;
        public Quaternion rotation;
        public Vector3 scale;
    }
}
