using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.Threading.Tasks;
using UnityEngine;
using UnityEngine.Networking;

namespace Collab.Unity
{
    /// <summary>
    /// Manages asset upload, download, and synchronization
    /// Handles prefabs, textures, audio, and other Unity assets
    /// </summary>
    public class CollabAssetManager : MonoBehaviour
    {
        [Header("Settings")]
        [SerializeField] private string assetCachePath = "";
        [SerializeField] private int maxConcurrentDownloads = 3;
        [SerializeField] private bool autoDownloadAssets = true;

        // Internal state
        private Dictionary<string, AssetMetadata> assetRegistry = new Dictionary<string, AssetMetadata>();
        private Queue<AssetDownloadRequest> downloadQueue = new Queue<AssetDownloadRequest>();
        private List<AssetDownloadRequest> activeDownloads = new List<AssetDownloadRequest>();
        private CollabNetworkClient networkClient;

        // Events
        public event Action<string, float> OnUploadProgress;
        public event Action<string> OnUploadComplete;
        public event Action<string, string> OnUploadError;
        public event Action<string, float> OnDownloadProgress;
        public event Action<string, UnityEngine.Object> OnDownloadComplete;
        public event Action<string, string> OnDownloadError;

        public void Initialize(CollabNetworkClient client)
        {
            networkClient = client;

            // Set cache path
            if (string.IsNullOrEmpty(assetCachePath))
            {
                assetCachePath = Path.Combine(Application.persistentDataPath, "CollabAssets");
            }

            // Create cache directory if it doesn't exist
            if (!Directory.Exists(assetCachePath))
            {
                Directory.CreateDirectory(assetCachePath);
            }

            Debug.Log($"[CollabAssetManager] Initialized with cache path: {assetCachePath}");
        }

        /// <summary>
        /// Upload a Unity asset to the collaboration server
        /// </summary>
        public async Task<string> UploadAsset(string filePath, string assetName, string assetType, string projectId)
        {
            if (networkClient == null)
            {
                OnUploadError?.Invoke(assetName, "Network client not initialized");
                return null;
            }

            if (!File.Exists(filePath))
            {
                OnUploadError?.Invoke(assetName, "File not found");
                return null;
            }

            try
            {
                Debug.Log($"[CollabAssetManager] Uploading asset: {assetName}");

                // Get upload URL from server
                var uploadRequest = new AssetUploadRequest
                {
                    name = assetName,
                    type = assetType,
                    project_id = projectId,
                    size = new FileInfo(filePath).Length
                };

                var uploadResponse = await networkClient.GetUploadUrl(uploadRequest);
                if (uploadResponse == null || string.IsNullOrEmpty(uploadResponse.upload_url))
                {
                    OnUploadError?.Invoke(assetName, "Failed to get upload URL");
                    return null;
                }

                // Read file bytes
                byte[] fileBytes = File.ReadAllBytes(filePath);

                // Upload to S3
                StartCoroutine(UploadToS3(uploadResponse.upload_url, fileBytes, assetName, uploadResponse.asset_id));

                return uploadResponse.asset_id;
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabAssetManager] Upload error: {ex.Message}");
                OnUploadError?.Invoke(assetName, ex.Message);
                return null;
            }
        }

        /// <summary>
        /// Upload bytes directly (for runtime-generated assets)
        /// </summary>
        public async Task<string> UploadBytes(byte[] data, string assetName, string assetType, string projectId)
        {
            if (networkClient == null)
            {
                OnUploadError?.Invoke(assetName, "Network client not initialized");
                return null;
            }

            try
            {
                Debug.Log($"[CollabAssetManager] Uploading bytes: {assetName}");

                // Get upload URL
                var uploadRequest = new AssetUploadRequest
                {
                    name = assetName,
                    type = assetType,
                    project_id = projectId,
                    size = data.Length
                };

                var uploadResponse = await networkClient.GetUploadUrl(uploadRequest);
                if (uploadResponse == null)
                {
                    OnUploadError?.Invoke(assetName, "Failed to get upload URL");
                    return null;
                }

                // Upload to S3
                StartCoroutine(UploadToS3(uploadResponse.upload_url, data, assetName, uploadResponse.asset_id));

                return uploadResponse.asset_id;
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabAssetManager] Upload error: {ex.Message}");
                OnUploadError?.Invoke(assetName, ex.Message);
                return null;
            }
        }

        /// <summary>
        /// Upload to S3 using presigned URL
        /// </summary>
        private IEnumerator UploadToS3(string uploadUrl, byte[] data, string assetName, string assetId)
        {
            using (UnityWebRequest request = UnityWebRequest.Put(uploadUrl, data))
            {
                request.method = "PUT";
                request.SetRequestHeader("Content-Type", "application/octet-stream");

                var operation = request.SendWebRequest();

                while (!operation.isDone)
                {
                    OnUploadProgress?.Invoke(assetName, operation.progress);
                    yield return null;
                }

                if (request.result == UnityWebRequest.Result.Success)
                {
                    Debug.Log($"[CollabAssetManager] Upload complete: {assetName}");
                    OnUploadComplete?.Invoke(assetId);

                    // Confirm upload with server
                    StartCoroutine(ConfirmUpload(assetId));
                }
                else
                {
                    Debug.LogError($"[CollabAssetManager] Upload failed: {request.error}");
                    OnUploadError?.Invoke(assetName, request.error);
                }
            }
        }

        /// <summary>
        /// Confirm asset upload with server
        /// </summary>
        private IEnumerator ConfirmUpload(string assetId)
        {
            yield return networkClient.ConfirmAssetUpload(assetId);
        }

        /// <summary>
        /// Download an asset from the server
        /// </summary>
        public void DownloadAsset(string assetId, string assetName, string assetType)
        {
            // Check if already in cache
            string cachePath = GetCachePath(assetId);
            if (File.Exists(cachePath))
            {
                Debug.Log($"[CollabAssetManager] Asset found in cache: {assetName}");
                LoadFromCache(assetId, assetName, assetType);
                return;
            }

            // Add to download queue
            var downloadRequest = new AssetDownloadRequest
            {
                assetId = assetId,
                assetName = assetName,
                assetType = assetType
            };

            downloadQueue.Enqueue(downloadRequest);
            ProcessDownloadQueue();
        }

        /// <summary>
        /// Process download queue
        /// </summary>
        private void ProcessDownloadQueue()
        {
            // Start downloads up to max concurrent limit
            while (downloadQueue.Count > 0 && activeDownloads.Count < maxConcurrentDownloads)
            {
                var request = downloadQueue.Dequeue();
                activeDownloads.Add(request);
                StartCoroutine(DownloadAssetCoroutine(request));
            }
        }

        /// <summary>
        /// Download asset coroutine
        /// </summary>
        private IEnumerator DownloadAssetCoroutine(AssetDownloadRequest request)
        {
            Debug.Log($"[CollabAssetManager] Downloading asset: {request.assetName}");

            // Get download URL from server
            Task<string> urlTask = networkClient.GetDownloadUrl(request.assetId);
            yield return new WaitUntil(() => urlTask.IsCompleted);

            if (urlTask.Result == null)
            {
                OnDownloadError?.Invoke(request.assetId, "Failed to get download URL");
                activeDownloads.Remove(request);
                ProcessDownloadQueue();
                yield break;
            }

            string downloadUrl = urlTask.Result;

            // Download the file
            using (UnityWebRequest www = UnityWebRequest.Get(downloadUrl))
            {
                var operation = www.SendWebRequest();

                while (!operation.isDone)
                {
                    OnDownloadProgress?.Invoke(request.assetId, operation.progress);
                    yield return null;
                }

                if (www.result == UnityWebRequest.Result.Success)
                {
                    // Save to cache
                    string cachePath = GetCachePath(request.assetId);
                    File.WriteAllBytes(cachePath, www.downloadHandler.data);

                    Debug.Log($"[CollabAssetManager] Download complete: {request.assetName}");

                    // Load the asset
                    LoadFromCache(request.assetId, request.assetName, request.assetType);
                }
                else
                {
                    Debug.LogError($"[CollabAssetManager] Download failed: {www.error}");
                    OnDownloadError?.Invoke(request.assetId, www.error);
                }
            }

            // Remove from active downloads and process queue
            activeDownloads.Remove(request);
            ProcessDownloadQueue();
        }

        /// <summary>
        /// Load asset from cache
        /// </summary>
        private void LoadFromCache(string assetId, string assetName, string assetType)
        {
            string cachePath = GetCachePath(assetId);

            if (!File.Exists(cachePath))
            {
                OnDownloadError?.Invoke(assetId, "Asset not found in cache");
                return;
            }

            byte[] data = File.ReadAllBytes(cachePath);

            // Load based on asset type
            UnityEngine.Object asset = null;

            switch (assetType.ToLower())
            {
                case "texture":
                case "image":
                    asset = LoadTexture(data);
                    break;

                case "audio":
                    // Audio requires special handling
                    Debug.LogWarning("[CollabAssetManager] Audio loading not yet implemented");
                    break;

                case "prefab":
                    // Prefabs require special handling
                    Debug.LogWarning("[CollabAssetManager] Prefab loading not yet implemented");
                    break;

                default:
                    Debug.LogWarning($"[CollabAssetManager] Unknown asset type: {assetType}");
                    break;
            }

            if (asset != null)
            {
                OnDownloadComplete?.Invoke(assetId, asset);

                // Add to registry
                assetRegistry[assetId] = new AssetMetadata
                {
                    id = assetId,
                    name = assetName,
                    type = assetType,
                    asset = asset
                };
            }
        }

        /// <summary>
        /// Load texture from bytes
        /// </summary>
        private Texture2D LoadTexture(byte[] data)
        {
            Texture2D texture = new Texture2D(2, 2);
            if (texture.LoadImage(data))
            {
                return texture;
            }
            return null;
        }

        /// <summary>
        /// Get cache file path for an asset
        /// </summary>
        private string GetCachePath(string assetId)
        {
            return Path.Combine(assetCachePath, assetId);
        }

        /// <summary>
        /// Get asset from registry
        /// </summary>
        public UnityEngine.Object GetAsset(string assetId)
        {
            if (assetRegistry.TryGetValue(assetId, out var metadata))
            {
                return metadata.asset;
            }
            return null;
        }

        /// <summary>
        /// Clear asset cache
        /// </summary>
        public void ClearCache()
        {
            try
            {
                if (Directory.Exists(assetCachePath))
                {
                    Directory.Delete(assetCachePath, true);
                    Directory.CreateDirectory(assetCachePath);
                }

                assetRegistry.Clear();
                Debug.Log("[CollabAssetManager] Cache cleared");
            }
            catch (Exception ex)
            {
                Debug.LogError($"[CollabAssetManager] Failed to clear cache: {ex.Message}");
            }
        }

        private void OnDestroy()
        {
            // Cancel all active downloads
            StopAllCoroutines();
            activeDownloads.Clear();
            downloadQueue.Clear();
        }
    }

    /// <summary>
    /// Asset metadata
    /// </summary>
    [Serializable]
    public class AssetMetadata
    {
        public string id;
        public string name;
        public string type;
        public UnityEngine.Object asset;
    }

    /// <summary>
    /// Asset download request
    /// </summary>
    public class AssetDownloadRequest
    {
        public string assetId;
        public string assetName;
        public string assetType;
    }

    /// <summary>
    /// Asset upload request
    /// </summary>
    [Serializable]
    public class AssetUploadRequest
    {
        public string name;
        public string type;
        public string project_id;
        public long size;
    }

    /// <summary>
    /// Asset upload response
    /// </summary>
    [Serializable]
    public class AssetUploadResponse
    {
        public string asset_id;
        public string upload_url;
    }
}
