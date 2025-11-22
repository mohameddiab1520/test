-- Unity Collaboration Platform - Seed Data for Development
-- This file contains sample data for local development and testing

-- Clear existing data (except system user)
DELETE FROM audit_logs WHERE user_id != '00000000-0000-0000-0000-000000000001';
DELETE FROM webhook_deliveries;
DELETE FROM webhooks;
DELETE FROM invitations;
DELETE FROM assets;
DELETE FROM session_participants;
DELETE FROM sessions;
DELETE FROM project_members WHERE user_id != '00000000-0000-0000-0000-000000000001';
DELETE FROM projects WHERE owner_id != '00000000-0000-0000-0000-000000000001';
DELETE FROM api_keys;
DELETE FROM refresh_tokens;
DELETE FROM users WHERE user_id != '00000000-0000-0000-0000-000000000001';

-- Insert test users
-- Password: "password123" (hashed with bcrypt)
-- You should replace this with actual bcrypt hashes in production
INSERT INTO users (user_id, email, password_hash, name, avatar_url, email_verified, status) VALUES
('11111111-1111-1111-1111-111111111111', 'alice@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Alice Johnson', 'https://i.pravatar.cc/150?img=1', TRUE, 'active'),
('22222222-2222-2222-2222-222222222222', 'bob@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Bob Smith', 'https://i.pravatar.cc/150?img=2', TRUE, 'active'),
('33333333-3333-3333-3333-333333333333', 'charlie@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Charlie Brown', 'https://i.pravatar.cc/150?img=3', TRUE, 'active'),
('44444444-4444-4444-4444-444444444444', 'diana@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Diana Prince', 'https://i.pravatar.cc/150?img=4', TRUE, 'active'),
('55555555-5555-5555-5555-555555555555', 'eve@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Eve Davis', 'https://i.pravatar.cc/150?img=5', FALSE, 'active');

-- Insert test projects
INSERT INTO projects (project_id, owner_id, name, description, unity_version, status, storage_limit) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'FPS Game Project', 'A multiplayer first-person shooter game built with Unity', '2022.3.15f1', 'active', 21474836480),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'Mobile Puzzle Game', 'Casual puzzle game for iOS and Android', '2022.3.10f1', 'active', 10737418240),
('cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', 'VR Experience', 'Virtual reality training simulation', '2023.1.5f1', 'active', 53687091200),
('dddddddd-dddd-dddd-dddd-dddddddddddd', '33333333-3333-3333-3333-333333333333', 'RPG Adventure', 'Open-world role-playing game', '2022.3.20f1', 'active', 107374182400);

-- Add project members
INSERT INTO project_members (project_id, user_id, role, invited_by) VALUES
-- FPS Game Project members
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'owner', NULL),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '22222222-2222-2222-2222-222222222222', 'developer', '11111111-1111-1111-1111-111111111111'),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '33333333-3333-3333-3333-333333333333', 'developer', '11111111-1111-1111-1111-111111111111'),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '44444444-4444-4444-4444-444444444444', 'viewer', '11111111-1111-1111-1111-111111111111'),

-- Mobile Puzzle Game members
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'owner', NULL),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', 'admin', '22222222-2222-2222-2222-222222222222'),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '55555555-5555-5555-5555-555555555555', 'developer', '22222222-2222-2222-2222-222222222222'),

-- VR Experience members
('cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', 'owner', NULL),
('cccccccc-cccc-cccc-cccc-cccccccccccc', '44444444-4444-4444-4444-444444444444', 'developer', '11111111-1111-1111-1111-111111111111'),

-- RPG Adventure members
('dddddddd-dddd-dddd-dddd-dddddddddddd', '33333333-3333-3333-3333-333333333333', 'owner', NULL),
('dddddddd-dddd-dddd-dddd-dddddddddddd', '11111111-1111-1111-1111-111111111111', 'developer', '33333333-3333-3333-3333-333333333333'),
('dddddddd-dddd-dddd-dddd-dddddddddddd', '22222222-2222-2222-2222-222222222222', 'developer', '33333333-3333-3333-3333-333333333333');

-- Insert test sessions
INSERT INTO sessions (session_id, project_id, owner_id, name, description, status, max_participants, is_public, voice_enabled, started_at) VALUES
('sess1111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'Level Design Session', 'Working on the multiplayer map design', 'active', 10, FALSE, TRUE, NOW() - INTERVAL '30 minutes'),
('sess2222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'UI Polish', 'Polishing the game UI and animations', 'active', 5, FALSE, TRUE, NOW() - INTERVAL '15 minutes'),
('sess3333-3333-3333-3333-333333333333', 'dddddddd-dddd-dddd-dddd-dddddddddddd', '33333333-3333-3333-3333-333333333333', 'Quest System Review', 'Reviewing the quest system implementation', 'paused', 8, FALSE, FALSE, NOW() - INTERVAL '2 hours');

-- Add session participants
INSERT INTO session_participants (session_id, user_id, role, status, current_scene, voice_muted) VALUES
-- Level Design Session
('sess1111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'owner', 'online', 'MainLevel', FALSE),
('sess1111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'editor', 'online', 'MainLevel', FALSE),
('sess1111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 'editor', 'away', 'MainLevel', TRUE),

-- UI Polish Session
('sess2222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 'owner', 'online', 'UIScene', FALSE),
('sess2222-2222-2222-2222-222222222222', '55555555-5555-5555-5555-555555555555', 'editor', 'online', 'UIScene', FALSE),

-- Quest System Review
('sess3333-3333-3333-3333-333333333333', '33333333-3333-3333-3333-333333333333', 'owner', 'offline', 'QuestScene', FALSE);

-- Insert test assets
INSERT INTO assets (asset_id, project_id, user_id, file_name, file_size, content_type, storage_path, cdn_url, category, tags, status, version_number) VALUES
-- FPS Game Project assets
('asset111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'player_model.fbx', 5242880, 'model/fbx', '/assets/fps/player_model.fbx', 'https://cdn.example.com/player_model.fbx', 'model', ARRAY['character', '3d', 'player'], 'ready', 1),
('asset222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '22222222-2222-2222-2222-222222222222', 'weapon_texture.png', 2097152, 'image/png', '/assets/fps/weapon_texture.png', 'https://cdn.example.com/weapon_texture.png', 'texture', ARRAY['weapon', 'texture'], 'ready', 2),
('asset333-3333-3333-3333-333333333333', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '33333333-3333-3333-3333-333333333333', 'level_music.mp3', 8388608, 'audio/mp3', '/assets/fps/level_music.mp3', 'https://cdn.example.com/level_music.mp3', 'audio', ARRAY['music', 'background'], 'ready', 1),

-- Mobile Puzzle Game assets
('asset444-4444-4444-4444-444444444444', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'ui_sprites.png', 1048576, 'image/png', '/assets/puzzle/ui_sprites.png', 'https://cdn.example.com/ui_sprites.png', 'sprite', ARRAY['ui', 'sprites'], 'ready', 3),
('asset555-5555-5555-5555-555555555555', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '55555555-5555-5555-5555-555555555555', 'puzzle_pieces.prefab', 524288, 'application/unity', '/assets/puzzle/puzzle_pieces.prefab', 'https://cdn.example.com/puzzle_pieces.prefab', 'prefab', ARRAY['puzzle', 'prefab'], 'ready', 1),

-- VR Experience assets
('asset666-6666-6666-6666-666666666666', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '11111111-1111-1111-1111-111111111111', 'vr_environment.scene', 15728640, 'application/unity', '/assets/vr/vr_environment.scene', 'https://cdn.example.com/vr_environment.scene', 'scene', ARRAY['vr', 'environment'], 'ready', 1),
('asset777-7777-7777-7777-777777777777', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '44444444-4444-4444-4444-444444444444', 'hand_models.fbx', 3145728, 'model/fbx', '/assets/vr/hand_models.fbx', 'https://cdn.example.com/hand_models.fbx', 'model', ARRAY['vr', 'hands', '3d'], 'ready', 2);

-- Update project storage usage
UPDATE projects p
SET storage_used = (
    SELECT COALESCE(SUM(a.file_size), 0)
    FROM assets a
    WHERE a.project_id = p.project_id AND a.status = 'ready'
);

-- Insert pending invitations
INSERT INTO invitations (project_id, invited_by, email, role, message, token, expires_at) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'newuser@example.com', 'developer', 'Join our FPS game development team!', 'token_abc123def456', NOW() + INTERVAL '7 days'),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'designer@example.com', 'viewer', 'Check out our puzzle game project', 'token_xyz789ghi012', NOW() + INTERVAL '7 days');

-- Insert sample webhooks
INSERT INTO webhooks (project_id, user_id, url, events, secret, active) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', 'https://webhook.site/test-fps', ARRAY['session.started', 'session.ended', 'asset.uploaded'], 'secret_fps_webhook', TRUE),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '22222222-2222-2222-2222-222222222222', 'https://webhook.site/test-puzzle', ARRAY['session.started', 'member.joined'], 'secret_puzzle_webhook', TRUE);

-- Insert sample audit logs
INSERT INTO audit_logs (user_id, project_id, action, resource_type, resource_id, ip_address, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'project.created', 'project', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '192.168.1.100', NOW() - INTERVAL '30 days'),
('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'member.added', 'project_member', '22222222-2222-2222-2222-222222222222', '192.168.1.100', NOW() - INTERVAL '25 days'),
('22222222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'project.created', 'project', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '192.168.1.101', NOW() - INTERVAL '20 days'),
('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'session.created', 'session', 'sess1111-1111-1111-1111-111111111111', '192.168.1.100', NOW() - INTERVAL '30 minutes'),
('22222222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'session.joined', 'session', 'sess1111-1111-1111-1111-111111111111', '192.168.1.101', NOW() - INTERVAL '28 minutes');

-- Success message
DO $$
DECLARE
    user_count INT;
    project_count INT;
    session_count INT;
    asset_count INT;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users WHERE user_id != '00000000-0000-0000-0000-000000000001';
    SELECT COUNT(*) INTO project_count FROM projects;
    SELECT COUNT(*) INTO session_count FROM sessions;
    SELECT COUNT(*) INTO asset_count FROM assets;

    RAISE NOTICE '=================================================';
    RAISE NOTICE 'Seed data loaded successfully!';
    RAISE NOTICE '=================================================';
    RAISE NOTICE 'Test Users: %', user_count;
    RAISE NOTICE 'Projects: %', project_count;
    RAISE NOTICE 'Active Sessions: %', (SELECT COUNT(*) FROM sessions WHERE status = 'active');
    RAISE NOTICE 'Assets: %', asset_count;
    RAISE NOTICE '=================================================';
    RAISE NOTICE 'Test Credentials:';
    RAISE NOTICE '  Email: alice@example.com';
    RAISE NOTICE '  Password: password123';
    RAISE NOTICE '=================================================';
END $$;
