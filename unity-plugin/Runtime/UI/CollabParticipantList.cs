using System;
using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;

namespace Collab.Unity.UI
{
    /// <summary>
    /// UI component that displays list of session participants
    /// Shows online status, voice activity, and user actions
    /// </summary>
    public class CollabParticipantList : MonoBehaviour
    {
        [Header("UI References")]
        [SerializeField] private RectTransform participantContainer;
        [SerializeField] private GameObject participantItemPrefab;
        [SerializeField] private Text sessionNameText;
        [SerializeField] private Text participantCountText;

        [Header("Settings")]
        [SerializeField] private bool showVoiceIndicators = true;
        [SerializeField] private bool showPresenceStatus = true;
        [SerializeField] private float updateInterval = 1f;

        // Internal state
        private Dictionary<string, ParticipantItem> participants = new Dictionary<string, ParticipantItem>();
        private float lastUpdateTime;

        private void Awake()
        {
            if (participantContainer == null)
            {
                Debug.LogError("[CollabParticipantList] Participant container not assigned!");
            }

            if (participantItemPrefab == null)
            {
                participantItemPrefab = CreateDefaultParticipantPrefab();
            }
        }

        private void OnEnable()
        {
            // Subscribe to events
            if (CollabManager.Instance != null)
            {
                var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
                if (networkClient != null)
                {
                    networkClient.OnUserJoined += HandleUserJoined;
                    networkClient.OnUserLeft += HandleUserLeft;
                }

                var voiceManager = CollabManager.Instance.GetComponent<CollabVoiceManager>();
                if (voiceManager != null)
                {
                    voiceManager.OnUserSpeaking += HandleUserSpeaking;
                    voiceManager.OnUserJoinedVoice += HandleUserJoinedVoice;
                    voiceManager.OnUserLeftVoice += HandleUserLeftVoice;
                }

                CollabManager.Instance.OnSessionJoined += HandleSessionJoined;
            }

            RefreshParticipantList();
        }

        private void OnDisable()
        {
            // Unsubscribe from events
            if (CollabManager.Instance != null)
            {
                var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
                if (networkClient != null)
                {
                    networkClient.OnUserJoined -= HandleUserJoined;
                    networkClient.OnUserLeft -= HandleUserLeft;
                }

                var voiceManager = CollabManager.Instance.GetComponent<CollabVoiceManager>();
                if (voiceManager != null)
                {
                    voiceManager.OnUserSpeaking -= HandleUserSpeaking;
                    voiceManager.OnUserJoinedVoice -= HandleUserJoinedVoice;
                    voiceManager.OnUserLeftVoice -= HandleUserLeftVoice;
                }

                CollabManager.Instance.OnSessionJoined -= HandleSessionJoined;
            }
        }

        private void Update()
        {
            if (Time.time - lastUpdateTime >= updateInterval)
            {
                UpdateParticipantList();
                lastUpdateTime = Time.time;
            }
        }

        /// <summary>
        /// Create default participant item prefab
        /// </summary>
        private GameObject CreateDefaultParticipantPrefab()
        {
            GameObject item = new GameObject("ParticipantItem");
            var rectTransform = item.AddComponent<RectTransform>();
            rectTransform.sizeDelta = new Vector2(250, 40);

            // Background
            var background = item.AddComponent<Image>();
            background.color = new Color(0.2f, 0.2f, 0.2f, 0.8f);

            // Horizontal layout
            var layout = item.AddComponent<HorizontalLayoutGroup>();
            layout.padding = new RectOffset(10, 10, 5, 5);
            layout.spacing = 10;
            layout.childAlignment = TextAnchor.MiddleLeft;
            layout.childForceExpandWidth = false;
            layout.childForceExpandHeight = false;
            layout.childControlWidth = false;
            layout.childControlHeight = false;

            // Status indicator
            GameObject statusObj = new GameObject("Status");
            statusObj.transform.SetParent(item.transform);
            var statusImage = statusObj.AddComponent<Image>();
            var statusRect = statusObj.GetComponent<RectTransform>();
            statusRect.sizeDelta = new Vector2(10, 10);

            // User name
            GameObject nameObj = new GameObject("Name");
            nameObj.transform.SetParent(item.transform);
            var nameText = nameObj.AddComponent<Text>();
            nameText.font = Resources.GetBuiltinResource<Font>("Arial.ttf");
            nameText.fontSize = 14;
            nameText.color = Color.white;
            nameText.alignment = TextAnchor.MiddleLeft;
            var nameRect = nameObj.GetComponent<RectTransform>();
            nameRect.sizeDelta = new Vector2(150, 30);

            // Voice indicator
            GameObject voiceObj = new GameObject("Voice");
            voiceObj.transform.SetParent(item.transform);
            var voiceImage = voiceObj.AddComponent<Image>();
            var voiceRect = voiceObj.GetComponent<RectTransform>();
            voiceRect.sizeDelta = new Vector2(20, 20);
            voiceObj.SetActive(false);

            return item;
        }

        /// <summary>
        /// Refresh entire participant list
        /// </summary>
        public void RefreshParticipantList()
        {
            // Clear existing participants
            ClearParticipantList();

            // Add current session participants
            if (CollabManager.Instance != null && CollabManager.Instance.IsConnected)
            {
                // TODO: Get participant list from server
                // For now, just add ourselves
                AddParticipant(CollabManager.Instance.CurrentUserId, "You", true);
            }

            UpdateParticipantCount();
        }

        /// <summary>
        /// Clear all participants from list
        /// </summary>
        private void ClearParticipantList()
        {
            foreach (var item in participants.Values)
            {
                if (item.gameObject != null)
                {
                    Destroy(item.gameObject);
                }
            }

            participants.Clear();
        }

        /// <summary>
        /// Add participant to list
        /// </summary>
        private void AddParticipant(string userId, string userName, bool isSelf = false)
        {
            if (participants.ContainsKey(userId))
            {
                Debug.LogWarning($"[CollabParticipantList] Participant already exists: {userId}");
                return;
            }

            GameObject itemObj = Instantiate(participantItemPrefab, participantContainer);
            itemObj.name = $"Participant_{userId}";

            var item = new ParticipantItem
            {
                userId = userId,
                userName = userName,
                isSelf = isSelf,
                gameObject = itemObj,
                statusIndicator = itemObj.transform.Find("Status")?.GetComponent<Image>(),
                nameText = itemObj.transform.Find("Name")?.GetComponent<Text>(),
                voiceIndicator = itemObj.transform.Find("Voice")?.gameObject
            };

            // Set name
            if (item.nameText != null)
            {
                item.nameText.text = isSelf ? $"{userName} (You)" : userName;
            }

            // Set initial status
            UpdateParticipantStatus(item, "online");

            participants[userId] = item;
            UpdateParticipantCount();

            Debug.Log($"[CollabParticipantList] Added participant: {userName}");
        }

        /// <summary>
        /// Remove participant from list
        /// </summary>
        private void RemoveParticipant(string userId)
        {
            if (participants.TryGetValue(userId, out ParticipantItem item))
            {
                if (item.gameObject != null)
                {
                    Destroy(item.gameObject);
                }

                participants.Remove(userId);
                UpdateParticipantCount();

                Debug.Log($"[CollabParticipantList] Removed participant: {item.userName}");
            }
        }

        /// <summary>
        /// Update participant list display
        /// </summary>
        private void UpdateParticipantList()
        {
            foreach (var item in participants.Values)
            {
                // Update status based on last activity
                float timeSinceActivity = Time.time - item.lastActivity;
                if (timeSinceActivity > 60f) // Inactive for 1 minute
                {
                    UpdateParticipantStatus(item, "away");
                }
                else if (timeSinceActivity > 300f) // Inactive for 5 minutes
                {
                    UpdateParticipantStatus(item, "offline");
                }
            }
        }

        /// <summary>
        /// Update participant status indicator
        /// </summary>
        private void UpdateParticipantStatus(ParticipantItem item, string status)
        {
            if (item.statusIndicator == null || !showPresenceStatus)
                return;

            item.status = status;

            Color statusColor = status switch
            {
                "online" => Color.green,
                "away" => Color.yellow,
                "busy" => Color.red,
                "offline" => Color.gray,
                _ => Color.white
            };

            item.statusIndicator.color = statusColor;
        }

        /// <summary>
        /// Update participant count display
        /// </summary>
        private void UpdateParticipantCount()
        {
            if (participantCountText != null)
            {
                participantCountText.text = $"Participants: {participants.Count}";
            }
        }

        /// <summary>
        /// Handle user joined event
        /// </summary>
        private void HandleUserJoined(string userId, string userName)
        {
            if (!participants.ContainsKey(userId))
            {
                AddParticipant(userId, userName, false);
            }
        }

        /// <summary>
        /// Handle user left event
        /// </summary>
        private void HandleUserLeft(string userId)
        {
            RemoveParticipant(userId);
        }

        /// <summary>
        /// Handle session joined
        /// </summary>
        private void HandleSessionJoined(string sessionId)
        {
            if (sessionNameText != null)
            {
                sessionNameText.text = $"Session: {sessionId}";
            }

            RefreshParticipantList();
        }

        /// <summary>
        /// Handle user speaking event
        /// </summary>
        private void HandleUserSpeaking(string userId, float volume)
        {
            if (participants.TryGetValue(userId, out ParticipantItem item))
            {
                if (showVoiceIndicators && item.voiceIndicator != null)
                {
                    item.voiceIndicator.SetActive(true);
                    item.lastActivity = Time.time;

                    // Animate voice indicator based on volume
                    var image = item.voiceIndicator.GetComponent<Image>();
                    if (image != null)
                    {
                        image.color = Color.Lerp(Color.gray, Color.green, volume);
                    }
                }
            }
        }

        /// <summary>
        /// Handle user joined voice
        /// </summary>
        private void HandleUserJoinedVoice(string userId)
        {
            if (participants.TryGetValue(userId, out ParticipantItem item))
            {
                item.isInVoice = true;

                if (item.voiceIndicator != null)
                {
                    item.voiceIndicator.SetActive(true);
                }
            }
        }

        /// <summary>
        /// Handle user left voice
        /// </summary>
        private void HandleUserLeftVoice(string userId)
        {
            if (participants.TryGetValue(userId, out ParticipantItem item))
            {
                item.isInVoice = false;

                if (item.voiceIndicator != null)
                {
                    item.voiceIndicator.SetActive(false);
                }
            }
        }

        /// <summary>
        /// Get participant by user ID
        /// </summary>
        public ParticipantItem GetParticipant(string userId)
        {
            participants.TryGetValue(userId, out ParticipantItem item);
            return item;
        }

        /// <summary>
        /// Get all participants
        /// </summary>
        public List<ParticipantItem> GetAllParticipants()
        {
            return new List<ParticipantItem>(participants.Values);
        }

        /// <summary>
        /// Show/hide voice indicators
        /// </summary>
        public void SetShowVoiceIndicators(bool show)
        {
            showVoiceIndicators = show;

            foreach (var item in participants.Values)
            {
                if (item.voiceIndicator != null)
                {
                    item.voiceIndicator.SetActive(show && item.isInVoice);
                }
            }
        }
    }

    /// <summary>
    /// Participant item data
    /// </summary>
    [Serializable]
    public class ParticipantItem
    {
        public string userId;
        public string userName;
        public bool isSelf;
        public string status = "online";
        public bool isInVoice;
        public float lastActivity;

        // UI References
        public GameObject gameObject;
        public Image statusIndicator;
        public Text nameText;
        public GameObject voiceIndicator;
    }
}
