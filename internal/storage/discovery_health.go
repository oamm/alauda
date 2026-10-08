package storage

// EffectiveHealthSQL uses si, d and hs aliases and is shared by discovery surfaces.
const EffectiveHealthSQL = `CASE WHEN si.enabled=0 THEN 'Disabled'
 WHEN d.health_enabled=0 OR NOT EXISTS (SELECT 1 FROM health_checks hc WHERE hc.instance_id=si.id AND hc.enabled=1 AND hc.deleted_at IS NULL AND ` + activeDiscoveryCheckSQL + `) THEN 'Unknown'
 WHEN julianday(hs.last_check_time) IS NULL OR julianday(hs.last_check_time)>julianday('now')+60/86400.0 OR julianday(hs.last_check_time)<julianday('now')-MAX(300,COALESCE((SELECT MAX(interval_seconds)*3 FROM health_checks hc WHERE hc.instance_id=si.id AND hc.enabled=1 AND hc.deleted_at IS NULL AND ` + activeDiscoveryCheckSQL + `),300))/86400.0 THEN 'Unknown'
 WHEN hs.current_state='HEALTH_STATE_HEALTHY' THEN 'Healthy'
 WHEN hs.current_state='HEALTH_STATE_UNHEALTHY' THEN 'Unhealthy'
 WHEN hs.current_state='HEALTH_STATE_DEGRADED' THEN 'Degraded'
 WHEN hs.current_state='HEALTH_STATE_DISABLED' THEN 'Disabled'
 ELSE 'Unknown' END`

const activeDiscoveryCheckSQL = `(hc.endpoint_id IS NULL OR hc.endpoint_id='' OR EXISTS (SELECT 1 FROM endpoints target WHERE target.id=hc.endpoint_id AND target.deleted_at IS NULL AND target.enabled=1)) AND (hc.type='HEALTH_CHECK_TYPE_DNS' OR COALESCE((SELECT target.port FROM endpoints target WHERE target.id=hc.endpoint_id),si.port) BETWEEN 1 AND 65535)`
