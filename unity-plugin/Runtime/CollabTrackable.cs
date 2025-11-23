using System;
using System.Collections.Generic;
using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Component that marks a GameObject for real-time collaboration tracking
    /// Automatically syncs transform and component changes
    /// </summary>
    [DisallowMultipleComponent]
    public class CollabTrackable : MonoBehaviour
    {
        [Header("Tracking Settings")]
        [SerializeField] private string trackableId;
        [SerializeField] private bool trackTransform = true;
        [SerializeField] private bool trackComponents = true;
        [SerializeField] private float syncRate = 30f; // Updates per second

        [Header("Transform Sync")]
        [SerializeField] private bool syncPosition = true;
        [SerializeField] private bool syncRotation = true;
        [SerializeField] private bool syncScale = true;

        [Header("Ownership")]
        [SerializeField] private string ownerId;
        [SerializeField] private bool isOwned = false;

        // Internal state
        private Vector3 lastPosition;
        private Quaternion lastRotation;
        private Vector3 lastScale;
        private float lastSyncTime;
        private float syncInterval;

        // Events
        public event Action<string> OnOwnershipChanged;
        public event Action<CollabTrackable> OnPropertyChanged;

        // Properties
        public string TrackableId => trackableId;
        public string OwnerId => ownerId;
        public bool IsOwned => isOwned;
        public bool TrackTransform => trackTransform;

        private void Awake()
        {
            // Generate unique ID if not set
            if (string.IsNullOrEmpty(trackableId))
            {
                trackableId = Guid.NewGuid().ToString();
            }

            syncInterval = 1f / syncRate;
            CaptureCurrentState();
        }

        private void OnEnable()
        {
            // Register with sync manager
            if (CollabManager.Instance != null)
            {
                var syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
                if (syncManager != null)
                {
                    syncManager.RegisterTrackable(this);
                }
            }
        }

        private void OnDisable()
        {
            // Unregister from sync manager
            if (CollabManager.Instance != null)
            {
                var syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
                if (syncManager != null)
                {
                    syncManager.UnregisterTrackable(this);
                }
            }
        }

        private void Update()
        {
            if (!isOwned || !trackTransform)
                return;

            // Check if enough time has passed for sync
            if (Time.time - lastSyncTime < syncInterval)
                return;

            // Check if transform has changed
            if (HasTransformChanged())
            {
                SyncTransform();
                CaptureCurrentState();
                lastSyncTime = Time.time;
            }
        }

        /// <summary>
        /// Check if transform has changed since last sync
        /// </summary>
        private bool HasTransformChanged()
        {
            bool changed = false;

            if (syncPosition)
            {
                changed |= Vector3.Distance(transform.position, lastPosition) > 0.001f;
            }

            if (syncRotation)
            {
                changed |= Quaternion.Angle(transform.rotation, lastRotation) > 0.1f;
            }

            if (syncScale)
            {
                changed |= Vector3.Distance(transform.localScale, lastScale) > 0.001f;
            }

            return changed;
        }

        /// <summary>
        /// Capture current transform state
        /// </summary>
        private void CaptureCurrentState()
        {
            lastPosition = transform.position;
            lastRotation = transform.rotation;
            lastScale = transform.localScale;
        }

        /// <summary>
        /// Sync transform to other clients
        /// </summary>
        private void SyncTransform()
        {
            if (CollabManager.Instance == null)
                return;

            var syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
            if (syncManager == null)
                return;

            // Create transform update operation
            var operation = new TransformOperation
            {
                trackableId = trackableId,
                position = syncPosition ? transform.position : lastPosition,
                rotation = syncRotation ? transform.rotation : lastRotation,
                scale = syncScale ? transform.localScale : lastScale,
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
            };

            syncManager.SendTransformUpdate(operation);
        }

        /// <summary>
        /// Apply received transform update
        /// </summary>
        public void ApplyTransformUpdate(TransformOperation operation)
        {
            if (isOwned)
                return; // Don't apply updates if we own this object

            if (syncPosition)
            {
                transform.position = operation.position;
            }

            if (syncRotation)
            {
                transform.rotation = operation.rotation;
            }

            if (syncScale)
            {
                transform.localScale = operation.scale;
            }

            CaptureCurrentState();
        }

        /// <summary>
        /// Request ownership of this trackable object
        /// </summary>
        public void RequestOwnership()
        {
            if (isOwned)
                return;

            if (CollabManager.Instance == null)
                return;

            var syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
            if (syncManager != null)
            {
                syncManager.RequestOwnership(trackableId);
            }
        }

        /// <summary>
        /// Release ownership of this trackable object
        /// </summary>
        public void ReleaseOwnership()
        {
            if (!isOwned)
                return;

            if (CollabManager.Instance == null)
                return;

            var syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
            if (syncManager != null)
            {
                syncManager.ReleaseOwnership(trackableId);
            }

            SetOwnership("", false);
        }

        /// <summary>
        /// Set ownership state
        /// </summary>
        public void SetOwnership(string newOwnerId, bool owned)
        {
            ownerId = newOwnerId;
            isOwned = owned;
            OnOwnershipChanged?.Invoke(ownerId);
        }

        /// <summary>
        /// Sync a component property change
        /// </summary>
        public void SyncProperty(string componentType, string propertyName, object value)
        {
            if (!isOwned || !trackComponents)
                return;

            if (CollabManager.Instance == null)
                return;

            var syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
            if (syncManager != null)
            {
                var operation = new PropertyOperation
                {
                    trackableId = trackableId,
                    componentType = componentType,
                    propertyName = propertyName,
                    value = value?.ToString() ?? "",
                    timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
                };

                syncManager.SendPropertyUpdate(operation);
            }

            OnPropertyChanged?.Invoke(this);
        }

        /// <summary>
        /// Apply received property update
        /// </summary>
        public void ApplyPropertyUpdate(PropertyOperation operation)
        {
            if (isOwned)
                return;

            // Find the component
            var component = GetComponent(operation.componentType);
            if (component == null)
                return;

            // Use reflection to set the property
            var property = component.GetType().GetProperty(operation.propertyName);
            if (property != null && property.CanWrite)
            {
                try
                {
                    var convertedValue = Convert.ChangeType(operation.value, property.PropertyType);
                    property.SetValue(component, convertedValue);
                }
                catch (Exception ex)
                {
                    Debug.LogWarning($"[CollabTrackable] Failed to set property {operation.propertyName}: {ex.Message}");
                }
            }

            OnPropertyChanged?.Invoke(this);
        }

        /// <summary>
        /// Get serialized state for network transmission
        /// </summary>
        public TrackableState GetState()
        {
            return new TrackableState
            {
                trackableId = trackableId,
                ownerId = ownerId,
                position = transform.position,
                rotation = transform.rotation,
                scale = transform.localScale,
                isActive = gameObject.activeSelf
            };
        }

        /// <summary>
        /// Apply received state
        /// </summary>
        public void ApplyState(TrackableState state)
        {
            if (isOwned)
                return;

            ownerId = state.ownerId;
            transform.position = state.position;
            transform.rotation = state.rotation;
            transform.localScale = state.scale;
            gameObject.SetActive(state.isActive);

            CaptureCurrentState();
        }

#if UNITY_EDITOR
        private void OnValidate()
        {
            // Ensure sync rate is reasonable
            syncRate = Mathf.Clamp(syncRate, 1f, 60f);
            syncInterval = 1f / syncRate;
        }

        private void OnDrawGizmos()
        {
            if (!trackTransform)
                return;

            // Draw ownership indicator
            Gizmos.color = isOwned ? Color.green : Color.yellow;
            Gizmos.DrawWireSphere(transform.position, 0.5f);
        }
#endif
    }

    /// <summary>
    /// Transform operation data for network transmission
    /// </summary>
    [Serializable]
    public class TransformOperation
    {
        public string trackableId;
        public Vector3 position;
        public Quaternion rotation;
        public Vector3 scale;
        public long timestamp;
    }

    /// <summary>
    /// Property operation data for network transmission
    /// </summary>
    [Serializable]
    public class PropertyOperation
    {
        public string trackableId;
        public string componentType;
        public string propertyName;
        public string value;
        public long timestamp;
    }

    /// <summary>
    /// Complete trackable state for synchronization
    /// </summary>
    [Serializable]
    public class TrackableState
    {
        public string trackableId;
        public string ownerId;
        public Vector3 position;
        public Quaternion rotation;
        public Vector3 scale;
        public bool isActive;
    }
}
