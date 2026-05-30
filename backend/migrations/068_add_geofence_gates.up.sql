ALTER TABLE projects
  ADD COLUMN geofence_gates JSON DEFAULT NULL COMMENT 'Array of {"name":"...","lat":0,"lng":0}',
  ADD COLUMN geofence_radius_meters INT UNSIGNED DEFAULT 100 COMMENT 'Project-wide geofence radius in meters';

-- Seed geofence gates for LG Display project (id=58)
UPDATE projects
SET geofence_gates = JSON_ARRAY(
  JSON_OBJECT('name', 'Cổng A', 'lat', 20.8628815, 'lng', 106.5653889),
  JSON_OBJECT('name', 'Cổng B', 'lat', 20.8666595, 'lng', 106.5666170),
  JSON_OBJECT('name', 'Cổng C', 'lat', 20.8679818, 'lng', 106.5711738),
  JSON_OBJECT('name', 'Cổng D', 'lat', 20.8648142, 'lng', 106.5705864)
),
geofence_radius_meters = 100
WHERE id = 58;
