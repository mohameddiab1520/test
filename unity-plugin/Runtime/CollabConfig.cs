using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Configuration settings for the Unity Collaboration Platform
    /// </summary>
    [CreateAssetMenu(fileName = "CollabConfig", menuName = "Collaboration/Config")]
    public class CollabConfig : ScriptableObject
    {
        [Header("Server Configuration")]
        [Tooltip("API Gateway URL")]
        public string apiUrl = "http://localhost:3000";

        [Tooltip("WebSocket URL for real-time sync")]
        public string wsUrl = "ws://localhost:8081";

        [Header("Authentication")]
        [Tooltip("User email for login")]
        public string email = "";

        [Tooltip("User password")]
        public string password = "";

        [Header("Session Settings")]
        [Tooltip("Auto-connect on start")]
        public bool autoConnect = false;

        [Tooltip("Session ID to join (leave empty to create new)")]
        public string sessionId = "";

        [Tooltip("Project ID")]
        public string projectId = "";

        [Header("Sync Settings")]
        [Tooltip("How often to send position updates (seconds)")]
        [Range(0.01f, 1f)]
        public float syncInterval = 0.05f; // 50ms = 20 updates per second

        [Tooltip("Enable voice chat")]
        public bool enableVoiceChat = false;

        [Tooltip("Enable presence indicators")]
        public bool enablePresence = true;

        [Header("Debug")]
        [Tooltip("Enable debug logging")]
        public bool debugMode = false;
    }
}
