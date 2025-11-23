using System;
using System.Collections;
using System.Collections.Generic;
using UnityEngine;

namespace Collab.Unity
{
    /// <summary>
    /// Manages voice chat for real-time collaboration
    /// Handles microphone input, audio streaming, and playback
    /// </summary>
    [RequireComponent(typeof(AudioSource))]
    public class CollabVoiceManager : MonoBehaviour
    {
        [Header("Settings")]
        [SerializeField] private bool autoStartMicrophone = false;
        [SerializeField] private int sampleRate = 16000;
        [SerializeField] private int recordLength = 1; // seconds
        [SerializeField] private float voiceThreshold = 0.01f;

        [Header("Audio Settings")]
        [SerializeField] private bool echoCancellation = true;
        [SerializeField] private bool noiseSuppression = true;
        [SerializeField] private float inputVolume = 1.0f;
        [SerializeField] private float outputVolume = 1.0f;

        [Header("Status")]
        [SerializeField] private bool isMicrophoneActive = false;
        [SerializeField] private bool isMuted = false;
        [SerializeField] private string currentMicrophone = "";

        // Internal state
        private AudioClip microphoneClip;
        private AudioSource audioSource;
        private Dictionary<string, AudioSource> remoteAudioSources = new Dictionary<string, AudioSource>();
        private CollabNetworkClient networkClient;
        private int lastMicrophonePosition = 0;
        private float[] audioBuffer;
        private Queue<AudioPacket> audioQueue = new Queue<AudioPacket>();

        // Events
        public event Action<bool> OnMicrophoneStateChanged;
        public event Action<bool> OnMuteStateChanged;
        public event Action<string, float> OnUserSpeaking;
        public event Action<string> OnUserJoinedVoice;
        public event Action<string> OnUserLeftVoice;

        // Properties
        public bool IsMicrophoneActive => isMicrophoneActive;
        public bool IsMuted => isMuted;
        public string[] AvailableMicrophones => Microphone.devices;

        private void Awake()
        {
            audioSource = GetComponent<AudioSource>();
            audioBuffer = new float[sampleRate * recordLength];
        }

        public void Initialize(CollabNetworkClient client)
        {
            networkClient = client;

            if (autoStartMicrophone && Microphone.devices.Length > 0)
            {
                StartMicrophone();
            }

            Debug.Log($"[CollabVoiceManager] Initialized with {Microphone.devices.Length} microphones available");
        }

        /// <summary>
        /// Start microphone recording
        /// </summary>
        public void StartMicrophone(string deviceName = null)
        {
            if (isMicrophoneActive)
            {
                Debug.LogWarning("[CollabVoiceManager] Microphone already active");
                return;
            }

            if (Microphone.devices.Length == 0)
            {
                Debug.LogError("[CollabVoiceManager] No microphones available");
                return;
            }

            // Use default microphone if none specified
            if (string.IsNullOrEmpty(deviceName))
            {
                deviceName = Microphone.devices[0];
            }

            currentMicrophone = deviceName;
            microphoneClip = Microphone.Start(deviceName, true, recordLength, sampleRate);

            if (microphoneClip == null)
            {
                Debug.LogError($"[CollabVoiceManager] Failed to start microphone: {deviceName}");
                return;
            }

            isMicrophoneActive = true;
            lastMicrophonePosition = 0;

            Debug.Log($"[CollabVoiceManager] Microphone started: {deviceName}");
            OnMicrophoneStateChanged?.Invoke(true);

            StartCoroutine(ProcessMicrophoneInput());
        }

        /// <summary>
        /// Stop microphone recording
        /// </summary>
        public void StopMicrophone()
        {
            if (!isMicrophoneActive)
                return;

            Microphone.End(currentMicrophone);
            isMicrophoneActive = false;

            Debug.Log("[CollabVoiceManager] Microphone stopped");
            OnMicrophoneStateChanged?.Invoke(false);

            StopCoroutine(ProcessMicrophoneInput());
        }

        /// <summary>
        /// Toggle mute state
        /// </summary>
        public void SetMuted(bool muted)
        {
            isMuted = muted;
            OnMuteStateChanged?.Invoke(isMuted);
            Debug.Log($"[CollabVoiceManager] Mute state: {isMuted}");
        }

        /// <summary>
        /// Process microphone input and send to network
        /// </summary>
        private IEnumerator ProcessMicrophoneInput()
        {
            while (isMicrophoneActive)
            {
                int currentPosition = Microphone.GetPosition(currentMicrophone);

                if (currentPosition < 0 || !Microphone.IsRecording(currentMicrophone))
                {
                    Debug.LogWarning("[CollabVoiceManager] Microphone recording stopped unexpectedly");
                    StopMicrophone();
                    yield break;
                }

                // Calculate samples to read
                int samplesAvailable = currentPosition - lastMicrophonePosition;
                if (samplesAvailable < 0)
                {
                    samplesAvailable += microphoneClip.samples;
                }

                // Read samples if enough are available
                if (samplesAvailable > sampleRate / 10) // Process every 100ms
                {
                    float[] samples = new float[samplesAvailable];
                    microphoneClip.GetData(samples, lastMicrophonePosition);

                    lastMicrophonePosition = currentPosition;

                    // Check if audio is above threshold (voice activity detection)
                    float volume = GetAudioVolume(samples);

                    if (!isMuted && volume > voiceThreshold)
                    {
                        // Apply input volume
                        if (inputVolume != 1.0f)
                        {
                            for (int i = 0; i < samples.Length; i++)
                            {
                                samples[i] *= inputVolume;
                            }
                        }

                        // Send audio packet to network
                        SendAudioPacket(samples);
                    }
                }

                yield return new WaitForSeconds(0.05f); // Check every 50ms
            }
        }

        /// <summary>
        /// Calculate audio volume (RMS)
        /// </summary>
        private float GetAudioVolume(float[] samples)
        {
            float sum = 0f;
            for (int i = 0; i < samples.Length; i++)
            {
                sum += samples[i] * samples[i];
            }
            return Mathf.Sqrt(sum / samples.Length);
        }

        /// <summary>
        /// Send audio packet to network
        /// </summary>
        private void SendAudioPacket(float[] samples)
        {
            if (networkClient == null)
                return;

            // Convert float samples to bytes
            byte[] bytes = new byte[samples.Length * 2]; // 16-bit audio
            for (int i = 0; i < samples.Length; i++)
            {
                short value = (short)(samples[i] * short.MaxValue);
                bytes[i * 2] = (byte)(value & 0xFF);
                bytes[i * 2 + 1] = (byte)((value >> 8) & 0xFF);
            }

            // Send via WebSocket
            var packet = new AudioPacket
            {
                userId = CollabManager.Instance.CurrentUserId,
                sessionId = CollabManager.Instance.CurrentSessionId,
                audioData = Convert.ToBase64String(bytes),
                sampleRate = sampleRate,
                timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds()
            };

            networkClient.SendAudioPacket(packet);
        }

        /// <summary>
        /// Receive and play audio packet from remote user
        /// </summary>
        public void ReceiveAudioPacket(AudioPacket packet)
        {
            if (packet.userId == CollabManager.Instance.CurrentUserId)
                return; // Don't play our own audio

            audioQueue.Enqueue(packet);

            // Trigger speaking event
            float volume = 0.5f; // Calculate actual volume if needed
            OnUserSpeaking?.Invoke(packet.userId, volume);

            // Process audio queue
            StartCoroutine(ProcessAudioQueue());
        }

        /// <summary>
        /// Process received audio queue
        /// </summary>
        private IEnumerator ProcessAudioQueue()
        {
            while (audioQueue.Count > 0)
            {
                var packet = audioQueue.Dequeue();

                // Get or create audio source for user
                if (!remoteAudioSources.TryGetValue(packet.userId, out AudioSource userAudioSource))
                {
                    GameObject audioObj = new GameObject($"Audio_{packet.userId}");
                    audioObj.transform.SetParent(transform);
                    userAudioSource = audioObj.AddComponent<AudioSource>();
                    userAudioSource.spatialBlend = 0; // 2D audio
                    userAudioSource.volume = outputVolume;
                    remoteAudioSources[packet.userId] = userAudioSource;
                }

                // Convert bytes back to float samples
                byte[] bytes = Convert.FromBase64String(packet.audioData);
                float[] samples = new float[bytes.Length / 2];

                for (int i = 0; i < samples.Length; i++)
                {
                    short value = (short)((bytes[i * 2 + 1] << 8) | bytes[i * 2]);
                    samples[i] = value / (float)short.MaxValue;
                }

                // Create audio clip and play
                AudioClip clip = AudioClip.Create($"Voice_{packet.userId}", samples.Length, 1, packet.sampleRate, false);
                clip.SetData(samples, 0);

                userAudioSource.clip = clip;
                userAudioSource.Play();

                yield return new WaitForSeconds(clip.length);
            }
        }

        /// <summary>
        /// User joined voice channel
        /// </summary>
        public void OnUserJoined(string userId)
        {
            Debug.Log($"[CollabVoiceManager] User joined voice: {userId}");
            OnUserJoinedVoice?.Invoke(userId);
        }

        /// <summary>
        /// User left voice channel
        /// </summary>
        public void OnUserLeft(string userId)
        {
            Debug.Log($"[CollabVoiceManager] User left voice: {userId}");
            OnUserLeftVoice?.Invoke(userId);

            // Cleanup audio source
            if (remoteAudioSources.TryGetValue(userId, out AudioSource audioSource))
            {
                Destroy(audioSource.gameObject);
                remoteAudioSources.Remove(userId);
            }
        }

        /// <summary>
        /// Set input volume
        /// </summary>
        public void SetInputVolume(float volume)
        {
            inputVolume = Mathf.Clamp01(volume);
        }

        /// <summary>
        /// Set output volume
        /// </summary>
        public void SetOutputVolume(float volume)
        {
            outputVolume = Mathf.Clamp01(volume);

            // Update all remote audio sources
            foreach (var audioSource in remoteAudioSources.Values)
            {
                audioSource.volume = outputVolume;
            }
        }

        /// <summary>
        /// Get list of active speakers
        /// </summary>
        public List<string> GetActiveSpeakers()
        {
            return new List<string>(remoteAudioSources.Keys);
        }

        private void OnDestroy()
        {
            StopMicrophone();

            // Cleanup all audio sources
            foreach (var audioSource in remoteAudioSources.Values)
            {
                if (audioSource != null)
                {
                    Destroy(audioSource.gameObject);
                }
            }

            remoteAudioSources.Clear();
        }

#if UNITY_EDITOR
        private void OnValidate()
        {
            inputVolume = Mathf.Clamp01(inputVolume);
            outputVolume = Mathf.Clamp01(outputVolume);
            sampleRate = Mathf.Clamp(sampleRate, 8000, 48000);
            voiceThreshold = Mathf.Clamp(voiceThreshold, 0.001f, 1f);
        }
#endif
    }

    /// <summary>
    /// Audio packet for network transmission
    /// </summary>
    [Serializable]
    public class AudioPacket
    {
        public string userId;
        public string sessionId;
        public string audioData; // Base64 encoded
        public int sampleRate;
        public long timestamp;
    }
}
