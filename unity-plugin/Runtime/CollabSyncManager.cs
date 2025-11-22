using System;
using System.Collections.Generic;
using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Manages synchronization of GameObjects across clients
    /// </summary>
    public class CollabSyncManager : MonoBehaviour
    {
        private CollabConfig config;
        private string currentSessionId;
        private bool isSyncing = false;
        private float syncTimer = 0f;

        private Dictionary<string, CollabSyncedObject> syncedObjects = new Dictionary<string, CollabSyncedObject>();
        private List<SyncOperation> pendingOperations = new List<SyncOperation>();

        public void Initialize(CollabConfig cfg)
        {
            config = cfg;
        }

        public void StartSync(string sessionId)
        {
            currentSessionId = sessionId;
            isSyncing = true;
            Log("Started syncing");

            // Find all synced objects in the scene
            RefreshSyncedObjects();

            // Subscribe to network events
            if (CollabManager.Instance != null)
            {
                var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
                if (networkClient != null)
                {
                    networkClient.OnMessage += HandleSyncMessage;
                }
            }
        }

        public void StopSync()
        {
            isSyncing = false;
            currentSessionId = "";
            Log("Stopped syncing");

            // Unsubscribe from network events
            if (CollabManager.Instance != null)
            {
                var networkClient = CollabManager.Instance.GetComponent<CollabNetworkClient>();
                if (networkClient != null)
                {
                    networkClient.OnMessage -= HandleSyncMessage;
                }
            }
        }

        /// <summary>
        /// Register a GameObject for synchronization
        /// </summary>
        public void RegisterObject(CollabSyncedObject syncedObject)
        {
            if (!syncedObjects.ContainsKey(syncedObject.ObjectId))
            {
                syncedObjects.Add(syncedObject.ObjectId, syncedObject);
                Log($"Registered object: {syncedObject.ObjectId}");
            }
        }

        /// <summary>
        /// Unregister a GameObject from synchronization
        /// </summary>
        public void UnregisterObject(string objectId)
        {
            if (syncedObjects.ContainsKey(objectId))
            {
                syncedObjects.Remove(objectId);
                Log($"Unregistered object: {objectId}");
            }
        }

        /// <summary>
        /// Send a sync operation to other clients
        /// </summary>
        public void SendOperation(SyncOperation operation)
        {
            if (!isSyncing) return;

            operation.sessionId = currentSessionId;
            operation.userId = CollabManager.Instance?.CurrentUserId;
            operation.timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();

            var message = new SyncMessage
            {
                type = "operation",
                sessionId = currentSessionId,
                operation = operation
            };

            string json = JsonUtility.ToJson(message);

            var networkClient = CollabManager.Instance?.GetComponent<CollabNetworkClient>();
            if (networkClient != null && networkClient.IsConnected)
            {
                networkClient.SendMessage(json);
                Log($"Sent operation: {operation.type} for {operation.objectId}");
            }
        }

        private void Update()
        {
            if (!isSyncing) return;

            syncTimer += Time.deltaTime;

            if (syncTimer >= config.syncInterval)
            {
                syncTimer = 0f;
                SyncObjects();
            }

            // Process pending operations
            ProcessPendingOperations();
        }

        private void SyncObjects()
        {
            foreach (var kvp in syncedObjects)
            {
                var syncedObject = kvp.Value;
                if (syncedObject != null && syncedObject.HasChanged())
                {
                    var operation = syncedObject.CreateSyncOperation();
                    SendOperation(operation);
                    syncedObject.MarkSynced();
                }
            }
        }

        private void HandleSyncMessage(string message)
        {
            try
            {
                var syncMessage = JsonUtility.FromJson<SyncMessage>(message);

                if (syncMessage.type == "sync" && syncMessage.operation != null)
                {
                    // Ignore our own operations
                    if (syncMessage.operation.userId == CollabManager.Instance?.CurrentUserId)
                    {
                        return;
                    }

                    pendingOperations.Add(syncMessage.operation);
                    Log($"Received operation: {syncMessage.operation.type} for {syncMessage.operation.objectId}");
                }
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabSyncManager] Failed to parse sync message: {ex.Message}");
            }
        }

        private void ProcessPendingOperations()
        {
            if (pendingOperations.Count == 0) return;

            foreach (var operation in pendingOperations)
            {
                ApplyOperation(operation);
            }

            pendingOperations.Clear();
        }

        private void ApplyOperation(SyncOperation operation)
        {
            if (syncedObjects.TryGetValue(operation.objectId, out var syncedObject))
            {
                syncedObject.ApplyOperation(operation);
            }
            else
            {
                Log($"Object not found: {operation.objectId}");
            }
        }

        private void RefreshSyncedObjects()
        {
            syncedObjects.Clear();

            var allSyncedObjects = FindObjectsOfType<CollabSyncedObject>();
            foreach (var obj in allSyncedObjects)
            {
                RegisterObject(obj);
            }

            Log($"Found {syncedObjects.Count} synced objects");
        }

        private void Log(string message)
        {
            if (config != null && config.debugMode)
            {
                Debug.Log($"[CollabSyncManager] {message}");
            }
        }
    }

    [Serializable]
    public class SyncMessage
    {
        public string type;
        public string sessionId;
        public SyncOperation operation;
    }

    [Serializable]
    public class SyncOperation
    {
        public string id;
        public string type; // "create", "update", "delete"
        public string objectId;
        public string userId;
        public string sessionId;
        public long timestamp;
        public TransformData transform;
        public string propertyPath;
        public string propertyValue;
    }

    [Serializable]
    public class TransformData
    {
        public Vector3Data position;
        public Vector3Data rotation;
        public Vector3Data scale;
    }

    [Serializable]
    public class Vector3Data
    {
        public float x;
        public float y;
        public float z;

        public Vector3Data(Vector3 v)
        {
            x = v.x;
            y = v.y;
            z = v.z;
        }

        public Vector3 ToVector3()
        {
            return new Vector3(x, y, z);
        }
    }
}
