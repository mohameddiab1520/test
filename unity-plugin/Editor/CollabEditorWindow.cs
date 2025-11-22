using UnityEngine;
using UnityEditor;

namespace Collab.Unity.Editor
{
    /// <summary>
    /// Editor window for managing collaboration sessions
    /// Accessible via Window > Collaboration
    /// </summary>
    public class CollabEditorWindow : EditorWindow
    {
        private CollabConfig config;
        private Vector2 scrollPosition;

        // UI State
        private string email = "";
        private string password = "";
        private string sessionName = "New Session";
        private string sessionId = "";
        private string projectId = "";

        private bool isConnected = false;
        private bool isInSession = false;
        private string statusMessage = "Not connected";

        [MenuItem("Window/Collaboration/Collaboration Manager")]
        public static void ShowWindow()
        {
            var window = GetWindow<CollabEditorWindow>("Collaboration");
            window.minSize = new Vector2(400, 500);
            window.Show();
        }

        private void OnEnable()
        {
            // Load config
            LoadConfig();

            // Subscribe to events
            if (CollabManager.Instance != null)
            {
                CollabManager.Instance.OnConnected += HandleConnected;
                CollabManager.Instance.OnDisconnected += HandleDisconnected;
                CollabManager.Instance.OnSessionJoined += HandleSessionJoined;
                CollabManager.Instance.OnError += HandleError;
            }
        }

        private void OnDisable()
        {
            // Unsubscribe from events
            if (CollabManager.Instance != null)
            {
                CollabManager.Instance.OnConnected -= HandleConnected;
                CollabManager.Instance.OnDisconnected -= HandleDisconnected;
                CollabManager.Instance.OnSessionJoined -= HandleSessionJoined;
                CollabManager.Instance.OnError -= HandleError;
            }
        }

        private void OnGUI()
        {
            scrollPosition = EditorGUILayout.BeginScrollView(scrollPosition);

            // Header
            DrawHeader();

            EditorGUILayout.Space(10);

            // Configuration Section
            DrawConfigurationSection();

            EditorGUILayout.Space(10);

            // Connection Section
            DrawConnectionSection();

            EditorGUILayout.Space(10);

            // Session Section
            if (isConnected)
            {
                DrawSessionSection();
            }

            EditorGUILayout.Space(10);

            // Status Section
            DrawStatusSection();

            EditorGUILayout.EndScrollView();
        }

        private void DrawHeader()
        {
            EditorGUILayout.BeginVertical(EditorStyles.helpBox);

            var headerStyle = new GUIStyle(EditorStyles.boldLabel)
            {
                fontSize = 18,
                alignment = TextAnchor.MiddleCenter
            };

            GUILayout.Label("Unity Collaboration Platform", headerStyle);
            GUILayout.Label("Real-time collaborative editing", EditorStyles.centeredGreyMiniLabel);

            EditorGUILayout.EndVertical();
        }

        private void DrawConfigurationSection()
        {
            EditorGUILayout.BeginVertical(EditorStyles.helpBox);
            EditorGUILayout.LabelField("Configuration", EditorStyles.boldLabel);

            config = (CollabConfig)EditorGUILayout.ObjectField("Config", config, typeof(CollabConfig), false);

            if (config == null)
            {
                EditorGUILayout.HelpBox("No configuration found. Create one via Assets > Create > Collaboration > Config", MessageType.Warning);

                if (GUILayout.Button("Create Config"))
                {
                    CreateConfig();
                }
            }
            else
            {
                EditorGUI.indentLevel++;
                EditorGUILayout.LabelField("API URL", config.apiUrl);
                EditorGUILayout.LabelField("WebSocket URL", config.wsUrl);
                EditorGUI.indentLevel--;
            }

            EditorGUILayout.EndVertical();
        }

        private void DrawConnectionSection()
        {
            EditorGUILayout.BeginVertical(EditorStyles.helpBox);
            EditorGUILayout.LabelField("Connection", EditorStyles.boldLabel);

            if (!isConnected)
            {
                email = EditorGUILayout.TextField("Email", email);
                password = EditorGUILayout.PasswordField("Password", password);

                EditorGUILayout.Space(5);

                GUI.enabled = !string.IsNullOrEmpty(email) && !string.IsNullOrEmpty(password) && config != null;

                if (GUILayout.Button("Connect", GUILayout.Height(30)))
                {
                    Connect();
                }

                GUI.enabled = true;
            }
            else
            {
                EditorGUILayout.HelpBox("Connected to collaboration server", MessageType.Info);

                if (GUILayout.Button("Disconnect", GUILayout.Height(30)))
                {
                    Disconnect();
                }
            }

            EditorGUILayout.EndVertical();
        }

        private void DrawSessionSection()
        {
            EditorGUILayout.BeginVertical(EditorStyles.helpBox);
            EditorGUILayout.LabelField("Session", EditorStyles.boldLabel);

            if (!isInSession)
            {
                // Create Session
                EditorGUILayout.LabelField("Create New Session", EditorStyles.miniBoldLabel);
                sessionName = EditorGUILayout.TextField("Session Name", sessionName);
                projectId = EditorGUILayout.TextField("Project ID", projectId);

                if (GUILayout.Button("Create Session", GUILayout.Height(25)))
                {
                    CreateSession();
                }

                EditorGUILayout.Space(10);

                // Join Session
                EditorGUILayout.LabelField("Join Existing Session", EditorStyles.miniBoldLabel);
                sessionId = EditorGUILayout.TextField("Session ID", sessionId);

                GUI.enabled = !string.IsNullOrEmpty(sessionId);

                if (GUILayout.Button("Join Session", GUILayout.Height(25)))
                {
                    JoinSession();
                }

                GUI.enabled = true;
            }
            else
            {
                EditorGUILayout.HelpBox($"In session: {CollabManager.Instance.CurrentSessionId}", MessageType.Info);

                EditorGUILayout.Space(5);

                // Session info
                EditorGUILayout.LabelField("Session ID", CollabManager.Instance.CurrentSessionId);

                EditorGUILayout.Space(5);

                if (GUILayout.Button("Leave Session", GUILayout.Height(30)))
                {
                    LeaveSession();
                }
            }

            EditorGUILayout.EndVertical();
        }

        private void DrawStatusSection()
        {
            EditorGUILayout.BeginVertical(EditorStyles.helpBox);
            EditorGUILayout.LabelField("Status", EditorStyles.boldLabel);

            var statusStyle = new GUIStyle(EditorStyles.label)
            {
                wordWrap = true
            };

            EditorGUILayout.LabelField(statusMessage, statusStyle);

            // Connection indicator
            var color = isConnected ? Color.green : Color.red;
            var rect = EditorGUILayout.GetControlRect(false, 20);
            EditorGUI.DrawRect(new Rect(rect.x, rect.y, 10, 10), color);
            EditorGUI.LabelField(new Rect(rect.x + 15, rect.y, rect.width - 15, rect.height),
                isConnected ? "Connected" : "Disconnected");

            EditorGUILayout.EndVertical();
        }

        private void LoadConfig()
        {
            // Try to find existing config
            var configs = Resources.FindObjectsOfTypeAll<CollabConfig>();
            if (configs.Length > 0)
            {
                config = configs[0];
            }
        }

        private void CreateConfig()
        {
            config = CreateInstance<CollabConfig>();
            AssetDatabase.CreateAsset(config, "Assets/CollabConfig.asset");
            AssetDatabase.SaveAssets();
            EditorGUIUtility.PingObject(config);
        }

        private void Connect()
        {
            if (config == null)
            {
                statusMessage = "No configuration set";
                return;
            }

            // Create CollabManager if doesn't exist
            if (CollabManager.Instance == null)
            {
                var go = new GameObject("CollabManager");
                go.AddComponent<CollabManager>();
            }

            // Update config with credentials
            config.email = email;
            config.password = password;

            statusMessage = "Connecting...";
            CollabManager.Instance.Connect();
        }

        private void Disconnect()
        {
            if (CollabManager.Instance != null)
            {
                CollabManager.Instance.Disconnect();
            }

            isConnected = false;
            isInSession = false;
            statusMessage = "Disconnected";
        }

        private void CreateSession()
        {
            if (CollabManager.Instance != null)
            {
                statusMessage = $"Creating session '{sessionName}'...";
                CollabManager.Instance.CreateSession(sessionName, projectId);
            }
        }

        private void JoinSession()
        {
            if (CollabManager.Instance != null)
            {
                statusMessage = $"Joining session '{sessionId}'...";
                CollabManager.Instance.JoinSession(sessionId);
            }
        }

        private void LeaveSession()
        {
            if (CollabManager.Instance != null)
            {
                statusMessage = "Leaving session...";
                CollabManager.Instance.LeaveSession();
                isInSession = false;
            }
        }

        private void HandleConnected()
        {
            isConnected = true;
            statusMessage = "Connected to collaboration server";
            Repaint();
        }

        private void HandleDisconnected()
        {
            isConnected = false;
            isInSession = false;
            statusMessage = "Disconnected from server";
            Repaint();
        }

        private void HandleSessionJoined(string sessionId)
        {
            isInSession = true;
            statusMessage = $"Joined session: {sessionId}";
            Repaint();
        }

        private void HandleError(string error)
        {
            statusMessage = $"Error: {error}";
            Repaint();
        }

        private void Update()
        {
            // Update status from manager
            if (CollabManager.Instance != null)
            {
                isConnected = CollabManager.Instance.IsConnected;
                isInSession = !string.IsNullOrEmpty(CollabManager.Instance.CurrentSessionId);
            }
        }
    }
}
