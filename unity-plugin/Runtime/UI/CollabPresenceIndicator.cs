using System;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;

namespace Collab.Unity.UI
{
    /// <summary>
    /// Visual indicator showing presence of other users in the scene
    /// Displays user cursors, selections, and activity status
    /// </summary>
    public class CollabPresenceIndicator : MonoBehaviour
    {
        [Header("Prefabs")]
        [SerializeField] private GameObject userCursorPrefab;
        [SerializeField] private GameObject selectionHighlightPrefab;

        [Header("Settings")]
        [SerializeField] private bool showUserCursors = true;
        [SerializeField] private bool showSelectionHighlights = true;
        [SerializeField] private bool showUserLabels = true;
        [SerializeField] private float cursorUpdateRate = 30f;

        [Header("Colors")]
        [SerializeField] private Color[] userColors = new Color[]
        {
            new Color(0.2f, 0.6f, 1f),    // Blue
            new Color(1f, 0.4f, 0.4f),    // Red
            new Color(0.4f, 1f, 0.4f),    // Green
            new Color(1f, 0.8f, 0.2f),    // Yellow
            new Color(1f, 0.4f, 1f),      // Magenta
            new Color(0.4f, 1f, 1f),      // Cyan
        };

        // Internal state
        private Dictionary<string, UserPresenceUI> userPresences = new Dictionary<string, UserPresenceUI>();
        private Canvas canvas;
        private float lastCursorUpdateTime;
        private float cursorUpdateInterval;

        private void Awake()
        {
            // Create canvas for UI elements
            CreateCanvas();
            cursorUpdateInterval = 1f / cursorUpdateRate;

            // Create default prefabs if none assigned
            if (userCursorPrefab == null)
            {
                userCursorPrefab = CreateDefaultCursorPrefab();
            }

            if (selectionHighlightPrefab == null)
            {
                selectionHighlightPrefab = CreateDefaultHighlightPrefab();
            }
        }

        private void OnEnable()
        {
            // Subscribe to presence events
            if (CollabManager.Instance != null)
            {
                var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
                if (networkClient != null)
                {
                    networkClient.OnPresenceUpdate += HandlePresenceUpdate;
                    networkClient.OnUserJoined += HandleUserJoined;
                    networkClient.OnUserLeft += HandleUserLeft;
                }
            }
        }

        private void OnDisable()
        {
            // Unsubscribe from events
            if (CollabManager.Instance != null)
            {
                var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
                if (networkClient != null)
                {
                    networkClient.OnPresenceUpdate -= HandlePresenceUpdate;
                    networkClient.OnUserJoined -= HandleUserJoined;
                    networkClient.OnUserLeft -= HandleUserLeft;
                }
            }
        }

        private void Update()
        {
            // Send our cursor position periodically
            if (Time.time - lastCursorUpdateTime >= cursorUpdateInterval)
            {
                SendCursorUpdate();
                lastCursorUpdateTime = Time.time;
            }
        }

        /// <summary>
        /// Create canvas for UI elements
        /// </summary>
        private void CreateCanvas()
        {
            GameObject canvasObj = new GameObject("PresenceCanvas");
            canvasObj.transform.SetParent(transform);

            canvas = canvasObj.AddComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceOverlay;
            canvas.sortingOrder = 100; // High priority

            var scaler = canvasObj.AddComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(1920, 1080);

            canvasObj.AddComponent<GraphicRaycaster>();
        }

        /// <summary>
        /// Create default cursor prefab
        /// </summary>
        private GameObject CreateDefaultCursorPrefab()
        {
            GameObject cursor = new GameObject("UserCursor");

            // Add image component
            var image = cursor.AddComponent<Image>();
            image.sprite = CreateCursorSprite();
            image.raycastTarget = false;

            // Add text label
            GameObject labelObj = new GameObject("Label");
            labelObj.transform.SetParent(cursor.transform);

            var text = labelObj.AddComponent<Text>();
            text.font = Resources.GetBuiltinResource<Font>("Arial.ttf");
            text.fontSize = 12;
            text.color = Color.white;
            text.alignment = TextAnchor.MiddleCenter;
            text.raycastTarget = false;

            var labelRect = labelObj.GetComponent<RectTransform>();
            labelRect.anchoredPosition = new Vector2(0, 20);
            labelRect.sizeDelta = new Vector2(100, 20);

            // Add shadow for better visibility
            var shadow = labelObj.AddComponent<Shadow>();
            shadow.effectColor = new Color(0, 0, 0, 0.5f);
            shadow.effectDistance = new Vector2(1, -1);

            return cursor;
        }

        /// <summary>
        /// Create cursor sprite
        /// </summary>
        private Sprite CreateCursorSprite()
        {
            // Create a simple cursor texture
            int size = 32;
            Texture2D texture = new Texture2D(size, size, TextureFormat.RGBA32, false);
            Color[] pixels = new Color[size * size];

            // Draw cursor shape
            for (int y = 0; y < size; y++)
            {
                for (int x = 0; x < size; x++)
                {
                    if (x < 10 && y < 20 && x <= y / 2)
                    {
                        pixels[y * size + x] = Color.white;
                    }
                    else
                    {
                        pixels[y * size + x] = Color.clear;
                    }
                }
            }

            texture.SetPixels(pixels);
            texture.Apply();

            return Sprite.Create(texture, new Rect(0, 0, size, size), new Vector2(0, 1));
        }

        /// <summary>
        /// Create default selection highlight prefab
        /// </summary>
        private GameObject CreateDefaultHighlightPrefab()
        {
            GameObject highlight = GameObject.CreatePrimitive(PrimitiveType.Cube);
            highlight.name = "SelectionHighlight";

            // Remove collider
            Destroy(highlight.GetComponent<Collider>());

            // Make it wireframe-like
            var renderer = highlight.GetComponent<Renderer>();
            var material = new Material(Shader.Find("Unlit/Color"));
            material.color = new Color(1, 1, 1, 0.3f);
            renderer.material = material;

            return highlight;
        }

        /// <summary>
        /// Handle presence update from network
        /// </summary>
        private void HandlePresenceUpdate(PresenceUpdate update)
        {
            if (update.userId == CollabManager.Instance.CurrentUserId)
                return; // Don't show our own cursor

            if (!userPresences.TryGetValue(update.userId, out UserPresenceUI presence))
            {
                // Create new presence UI
                presence = CreateUserPresence(update.userId, update.userName);
                userPresences[update.userId] = presence;
            }

            // Update cursor position
            if (showUserCursors && presence.cursor != null)
            {
                var rectTransform = presence.cursor.GetComponent<RectTransform>();
                rectTransform.anchoredPosition = new Vector2(update.cursorX, update.cursorY);
            }

            // Update selection highlight
            if (showSelectionHighlights && !string.IsNullOrEmpty(update.selectedObjectId))
            {
                UpdateSelectionHighlight(presence, update.selectedObjectId);
            }
            else if (presence.highlight != null)
            {
                presence.highlight.SetActive(false);
            }

            // Update activity status
            presence.lastActivity = Time.time;
            presence.status = update.status;
        }

        /// <summary>
        /// Handle user joined
        /// </summary>
        private void HandleUserJoined(string userId, string userName)
        {
            if (userId == CollabManager.Instance.CurrentUserId)
                return;

            if (!userPresences.ContainsKey(userId))
            {
                var presence = CreateUserPresence(userId, userName);
                userPresences[userId] = presence;
                Debug.Log($"[PresenceIndicator] User joined: {userName}");
            }
        }

        /// <summary>
        /// Handle user left
        /// </summary>
        private void HandleUserLeft(string userId)
        {
            if (userPresences.TryGetValue(userId, out UserPresenceUI presence))
            {
                // Destroy UI elements
                if (presence.cursor != null)
                    Destroy(presence.cursor);

                if (presence.highlight != null)
                    Destroy(presence.highlight);

                userPresences.Remove(userId);
                Debug.Log($"[PresenceIndicator] User left: {presence.userName}");
            }
        }

        /// <summary>
        /// Create user presence UI
        /// </summary>
        private UserPresenceUI CreateUserPresence(string userId, string userName)
        {
            var presence = new UserPresenceUI
            {
                userId = userId,
                userName = userName,
                color = GetUserColor(userId)
            };

            // Create cursor
            if (showUserCursors && userCursorPrefab != null)
            {
                presence.cursor = Instantiate(userCursorPrefab, canvas.transform);
                presence.cursor.name = $"Cursor_{userId}";

                // Set color
                var image = presence.cursor.GetComponent<Image>();
                if (image != null)
                {
                    image.color = presence.color;
                }

                // Set label
                var label = presence.cursor.GetComponentInChildren<Text>();
                if (label != null && showUserLabels)
                {
                    label.text = userName;
                    label.color = presence.color;
                }
            }

            // Create selection highlight
            if (showSelectionHighlights && selectionHighlightPrefab != null)
            {
                presence.highlight = Instantiate(selectionHighlightPrefab);
                presence.highlight.name = $"Highlight_{userId}";
                presence.highlight.SetActive(false);

                // Set color
                var renderer = presence.highlight.GetComponent<Renderer>();
                if (renderer != null)
                {
                    var material = new Material(renderer.material);
                    material.color = new Color(presence.color.r, presence.color.g, presence.color.b, 0.3f);
                    renderer.material = material;
                }
            }

            return presence;
        }

        /// <summary>
        /// Update selection highlight for a user
        /// </summary>
        private void UpdateSelectionHighlight(UserPresenceUI presence, string objectId)
        {
            if (presence.highlight == null)
                return;

            // Find the trackable object
            var trackables = FindObjectsOfType<CollabTrackable>();
            foreach (var trackable in trackables)
            {
                if (trackable.TrackableId == objectId)
                {
                    presence.highlight.SetActive(true);
                    presence.highlight.transform.position = trackable.transform.position;
                    presence.highlight.transform.rotation = trackable.transform.rotation;
                    presence.highlight.transform.localScale = trackable.transform.localScale * 1.1f; // Slightly larger
                    return;
                }
            }

            // Object not found, hide highlight
            presence.highlight.SetActive(false);
        }

        /// <summary>
        /// Get color for user (consistent based on ID)
        /// </summary>
        private Color GetUserColor(string userId)
        {
            int hash = userId.GetHashCode();
            int index = Math.Abs(hash) % userColors.Length;
            return userColors[index];
        }

        /// <summary>
        /// Send cursor update to network
        /// </summary>
        private void SendCursorUpdate()
        {
            if (CollabManager.Instance == null || !CollabManager.Instance.IsConnected)
                return;

            var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
            if (networkClient == null)
                return;

            // Get mouse position
            Vector2 mousePos = Input.mousePosition;

            // Get selected object (if any)
            string selectedObjectId = GetSelectedObjectId();

            var update = new PresenceUpdate
            {
                userId = CollabManager.Instance.CurrentUserId,
                userName = "Me", // Should get from user profile
                cursorX = mousePos.x,
                cursorY = mousePos.y,
                selectedObjectId = selectedObjectId,
                status = "active",
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
            };

            networkClient.SendPresenceUpdate(update);
        }

        /// <summary>
        /// Get currently selected object ID
        /// </summary>
        private string GetSelectedObjectId()
        {
            // Check if there's a selected trackable object
            // This would typically integrate with Unity's selection system
            // For now, return empty string
            return "";
        }

        /// <summary>
        /// Get list of active users
        /// </summary>
        public List<UserPresenceUI> GetActiveUsers()
        {
            return new List<UserPresenceUI>(userPresences.Values);
        }

        /// <summary>
        /// Show/hide user cursors
        /// </summary>
        public void SetShowCursors(bool show)
        {
            showUserCursors = show;

            foreach (var presence in userPresences.Values)
            {
                if (presence.cursor != null)
                {
                    presence.cursor.SetActive(show);
                }
            }
        }

        /// <summary>
        /// Show/hide selection highlights
        /// </summary>
        public void SetShowHighlights(bool show)
        {
            showSelectionHighlights = show;

            foreach (var presence in userPresences.Values)
            {
                if (presence.highlight != null)
                {
                    presence.highlight.SetActive(show && !string.IsNullOrEmpty(presence.selectedObjectId));
                }
            }
        }

        private void OnDestroy()
        {
            // Cleanup all presence UI
            foreach (var presence in userPresences.Values)
            {
                if (presence.cursor != null)
                    Destroy(presence.cursor);

                if (presence.highlight != null)
                    Destroy(presence.highlight);
            }

            userPresences.Clear();
        }
    }

    /// <summary>
    /// User presence UI data
    /// </summary>
    [Serializable]
    public class UserPresenceUI
    {
        public string userId;
        public string userName;
        public Color color;
        public GameObject cursor;
        public GameObject highlight;
        public string selectedObjectId;
        public string status;
        public float lastActivity;
    }

    /// <summary>
    /// Presence update data for network transmission
    /// </summary>
    [Serializable]
    public class PresenceUpdate
    {
        public string userId;
        public string userName;
        public float cursorX;
        public float cursorY;
        public string selectedObjectId;
        public string status;
        public long timestamp;
    }
}
