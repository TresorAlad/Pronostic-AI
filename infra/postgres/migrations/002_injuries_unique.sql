-- Dedupe injuries: one active row per player/team/type
DELETE FROM injuries a
USING injuries b
WHERE a.id > b.id
  AND a.player_id = b.player_id
  AND a.team_id = b.team_id
  AND COALESCE(a.type, '') = COALESCE(b.type, '');

ALTER TABLE injuries
ADD CONSTRAINT injuries_player_team_type_unique UNIQUE (player_id, team_id, type);
