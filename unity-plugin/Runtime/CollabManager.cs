using System;
using System.Collections.Generic;
using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Main manager for the Unity Collaboration Platform
    /// Singleton pattern to manage connection and synchronization
    /// </summary>
    public class CollabManager : MonoBehaviour
    {
        private static CollabManager _instance;
        public static CollabManager Instance
        {
            get
            {
                if (_instance == null)
                {
                    GameObject go = new GameObject("CollabManager");
                    _instance = go.AddComponent<CollabManager>();
                    DontDestroyOnLoad(go);
                }
                return _instance;
            }
        }

        [Header("Configuration")]
        [SerializeField] private CollabConfig config;

        [Header("Status")]
        [SerializeField] private bool isConnected = false;
        [SerializeField] private string currentSessionId = "";
        [SerializeField] private string currentUserId = "";

        // Components
        private CollabNetworkClient networkClient;
        private CollabSyncManager syncManager;

        // Events
        public event Action OnConnected;
        public event Action OnDisconnected;
        public event Action<string> OnSessionJoined;
        public event Action<string> OnError;

        // Properties
        public bool IsConnected => isConnected;
        public string CurrentSessionId => currentSessionId;
        public string CurrentUserId => currentUserId;
        public CollabConfig Config => config;

        private void Awake()
        {
            if (_instance != null && _instance != this)
            {
                Destroy(gameObject);
                return;
            }

            _instance = this;
            DontDestroyOnLoad(gameObject);

            Initialize();
        }

        private void Initialize()
        {
            if (config == null)
            {
                Debug.LogError("[CollabManager] No configuration found! Please assign a CollabConfig.");
                return;
            }

            // Initialize network client
            networkClient = gameObject.AddComponent<CollabNetworkClient>();
            networkClient.Initialize(config);

            // Initialize sync manager
            syncManager = gameObject.AddComponent<CollabSyncManager>();
            syncManager.Initialize(config);

            // Subscribe to events
            networkClient.OnConnected += HandleConnected;
            networkClient.OnDisconnected += HandleDisconnected;
            networkClient.OnError += HandleError;

            if (config.autoConnect)
            {
                Connect();
            }

            Log("CollabManager initialized");
        }

        private void Start()
        {
            if (config != null && config.autoConnect)
            {
                Connect();
            }
        }

        /// <summary>
        /// Connect to the collaboration server
        /// </summary>
        public async void Connect()
        {
            if (isConnected)
            {
                Debug.LogWarning("[CollabManager] Already connected");
                return;
            }

            try
            {
                Log("Connecting to collaboration server...");

                // Authenticate first
                var authResult = await networkClient.Authenticate(config.email, config.password);
                if (authResult)
                {
                    currentUserId = networkClient.UserId;
                    Log($"Authenticated as user: {currentUserId}");

                    // Connect WebSocket
                    await networkClient.ConnectWebSocket();
                }
                else
                {
                    OnError?.Invoke("Authentication failed");
                }
            }
            catch (Exception ex)
            {
                HandleError(ex.Message);
            }
        }

        /// <summary>
        /// Disconnect from the collaboration server
        /// </summary>
        public void Disconnect()
        {
            if (!isConnected)
            {
                return;
            }

            Log("Disconnecting from collaboration server...");
            networkClient.Disconnect();

            if (syncManager != null)
            {
                syncManager.StopSync();
            }

            currentSessionId = "";
            isConnected = false;
        }

        /// <summary>
        /// Create a new collaboration session
        /// </summary>
        public async void CreateSession(string sessionName, string projectId)
        {
            if (!isConnected)
            {
                OnError?.Invoke("Not connected to server");
                return;
            }

            try
            {
                Log($"Creating session: {sessionName}");
                var session = await networkClient.CreateSession(sessionName, projectId);

                if (session != null)
                {
                    currentSessionId = session.id;
                    OnSessionJoined?.Invoke(currentSessionId);

                    // Start syncing
                    syncManager.StartSync(currentSessionId);

                    Log($"Session created: {currentSessionId}");
                }
            }
            catch (Exception ex)
            {
                HandleError($"Failed to create session: {ex.Message}");
            }
        }

        /// <summary>
        /// Join an existing collaboration session
        /// </summary>
        public async void JoinSession(string sessionId)
        {
            if (!isConnected)
            {
                OnError?.Invoke("Not connected to server");
                return;
            }

            try
            {
                Log($"Joining session: {sessionId}");
                var success = await networkClient.JoinSession(sessionId);

                if (success)
                {
                    currentSessionId = sessionId;
                    OnSessionJoined?.Invoke(currentSessionId);

                    // Start syncing
                    syncManager.StartSync(currentSessionId);

                    Log($"Joined session: {currentSessionId}");
                }
            }
            catch (Exception ex)
            {
                HandleError($"Failed to join session: {ex.Message}");
            }
        }

        /// <summary>
        /// Leave the current session
        /// </summary>
        public async void LeaveSession()
        {
            if (string.IsNullOrEmpty(currentSessionId))
            {
                return;
            }

            try
            {
                Log("Leaving session...");
                await networkClient.LeaveSession(currentSessionId);

                syncManager.StopSync();
                currentSessionId = "";

                Log("Left session");
            }
            catch (Exception ex)
            {
                HandleError($"Failed to leave session: {ex.Message}");
            }
        }

        private void HandleConnected()
        {
            isConnected = true;
            Log("Connected to collaboration server");
            OnConnected?.Invoke();
        }

        private void HandleDisconnected()
        {
            isConnected = false;
            currentSessionId = "";
            Log("Disconnected from collaboration server");
            OnDisconnected?.Invoke();
        }

        private void HandleError(string error)
        {
            Debug.LogError($"[CollabManager] Error: {error}");
            OnError?.Invoke(error);
        }

        private void Log(string message)
        {
            if (config != null && config.debugMode)
            {
                Debug.Log($"[CollabManager] {message}");
            }
        }

        private void OnDestroy()
        {
            if (networkClient != null)
            {
                networkClient.OnConnected -= HandleConnected;
                networkClient.OnDisconnected -= HandleDisconnected;
                networkClient.OnError -= HandleError;
            }

            Disconnect();
        }
    }
}
