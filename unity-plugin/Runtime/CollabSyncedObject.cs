using System;
using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Component that marks a GameObject for synchronization across clients
    /// Attach this to any GameObject you want to sync
    /// </summary>
    [AddComponentMenu("Collaboration/Synced Object")]
    public class CollabSyncedObject : MonoBehaviour
    {
        [Header("Sync Settings")]
        [Tooltip("Unique ID for this object (auto-generated if empty)")]
        [SerializeField] private string objectId;

        [Tooltip("Is this object owned by the local user?")]
        [SerializeField] private bool isLocalOwner = true;

        [Header("What to Sync")]
        [SerializeField] private bool syncPosition = true;
        [SerializeField] private bool syncRotation = true;
        [SerializeField] private bool syncScale = false;

        [Header("Interpolation")]
        [Tooltip("Smooth remote updates")]
        [SerializeField] private bool useInterpolation = true;

        [SerializeField] private float interpolationSpeed = 10f;

        // Cached transform data
        private Vector3 lastPosition;
        private Quaternion lastRotation;
        private Vector3 lastScale;

        // Remote target data (for interpolation)
        private Vector3 targetPosition;
        private Quaternion targetRotation;
        private Vector3 targetScale;

        private bool hasChanged = false;
        private CollabSyncManager syncManager;

        public string ObjectId => objectId;
        public bool IsLocalOwner => isLocalOwner;

        private void Awake()
        {
            // Generate unique ID if not set
            if (string.IsNullOrEmpty(objectId))
            {
                objectId = $"{gameObject.name}_{Guid.NewGuid().ToString("N").Substring(0, 8)}";
            }

            // Cache initial transform
            CacheTransform();
            targetPosition = transform.position;
            targetRotation = transform.rotation;
            targetScale = transform.localScale;
        }

        private void Start()
        {
            // Register with sync manager
            if (CollabManager.Instance != null)
            {
                syncManager = CollabManager.Instance.GetComponent<CollabSyncManager>();
                if (syncManager != null)
                {
                    syncManager.RegisterObject(this);
                }
            }
        }

        private void Update()
        {
            if (isLocalOwner)
            {
                // Check for changes
                CheckForChanges();
            }
            else if (useInterpolation)
            {
                // Interpolate to target transform
                InterpolateTransform();
            }
        }

        /// <summary>
        /// Check if the transform has changed since last sync
        /// </summary>
        public bool HasChanged()
        {
            return hasChanged;
        }

        /// <summary>
        /// Mark the object as synced (reset changed flag)
        /// </summary>
        public void MarkSynced()
        {
            hasChanged = false;
            CacheTransform();
        }

        /// <summary>
        /// Create a sync operation from current state
        /// </summary>
        public SyncOperation CreateSyncOperation()
        {
            var operation = new SyncOperation
            {
                id = Guid.NewGuid().ToString(),
                type = "update",
                objectId = objectId,
                transform = new TransformData
                {
                    position = syncPosition ? new Vector3Data(transform.position) : null,
                    rotation = syncRotation ? new Vector3Data(transform.rotation.eulerAngles) : null,
                    scale = syncScale ? new Vector3Data(transform.localScale) : null
                }
            };

            return operation;
        }

        /// <summary>
        /// Apply a sync operation received from remote
        /// </summary>
        public void ApplyOperation(SyncOperation operation)
        {
            if (operation == null || operation.transform == null)
            {
                return;
            }

            // Don't apply if we own this object
            if (isLocalOwner)
            {
                return;
            }

            if (useInterpolation)
            {
                // Set target for interpolation
                if (operation.transform.position != null && syncPosition)
                {
                    targetPosition = operation.transform.position.ToVector3();
                }

                if (operation.transform.rotation != null && syncRotation)
                {
                    targetRotation = Quaternion.Euler(operation.transform.rotation.ToVector3());
                }

                if (operation.transform.scale != null && syncScale)
                {
                    targetScale = operation.transform.scale.ToVector3();
                }
            }
            else
            {
                // Apply directly
                if (operation.transform.position != null && syncPosition)
                {
                    transform.position = operation.transform.position.ToVector3();
                }

                if (operation.transform.rotation != null && syncRotation)
                {
                    transform.rotation = Quaternion.Euler(operation.transform.rotation.ToVector3());
                }

                if (operation.transform.scale != null && syncScale)
                {
                    transform.localScale = operation.transform.scale.ToVector3();
                }
            }
        }

        /// <summary>
        /// Transfer ownership of this object
        /// </summary>
        public void SetOwnership(bool isOwner)
        {
            isLocalOwner = isOwner;

            if (isOwner)
            {
                // Reset targets to current transform
                targetPosition = transform.position;
                targetRotation = transform.rotation;
                targetScale = transform.localScale;
                CacheTransform();
            }
        }

        private void CheckForChanges()
        {
            bool changed = false;

            if (syncPosition && Vector3.Distance(transform.position, lastPosition) > 0.001f)
            {
                changed = true;
            }

            if (syncRotation && Quaternion.Angle(transform.rotation, lastRotation) > 0.1f)
            {
                changed = true;
            }

            if (syncScale && Vector3.Distance(transform.localScale, lastScale) > 0.001f)
            {
                changed = true;
            }

            hasChanged = changed;
        }

        private void CacheTransform()
        {
            lastPosition = transform.position;
            lastRotation = transform.rotation;
            lastScale = transform.localScale;
        }

        private void InterpolateTransform()
        {
            if (syncPosition)
            {
                transform.position = Vector3.Lerp(
                    transform.position,
                    targetPosition,
                    Time.deltaTime * interpolationSpeed
                );
            }

            if (syncRotation)
            {
                transform.rotation = Quaternion.Slerp(
                    transform.rotation,
                    targetRotation,
                    Time.deltaTime * interpolationSpeed
                );
            }

            if (syncScale)
            {
                transform.localScale = Vector3.Lerp(
                    transform.localScale,
                    targetScale,
                    Time.deltaTime * interpolationSpeed
                );
            }
        }

        private void OnDestroy()
        {
            // Unregister from sync manager
            if (syncManager != null)
            {
                syncManager.UnregisterObject(objectId);
            }

            // Send delete operation if we own this object
            if (isLocalOwner && CollabManager.Instance != null && CollabManager.Instance.IsConnected)
            {
                var operation = new SyncOperation
                {
                    id = Guid.NewGuid().ToString(),
                    type = "delete",
                    objectId = objectId
                };

                syncManager?.SendOperation(operation);
            }
        }

        private void OnDrawGizmosSelected()
        {
            // Visualize ownership
            Gizmos.color = isLocalOwner ? Color.green : Color.yellow;
            Gizmos.DrawWireSphere(transform.position, 0.5f);
        }
    }
}
