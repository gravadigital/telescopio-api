-- Actualizar eventos que quedaron con stages antiguos
UPDATE events 
SET stage = 'participation' 
WHERE stage::text IN ('registration', 'attachment_upload');

-- Verificar que no haya más valores inválidos
SELECT stage, COUNT(*) 
FROM events 
GROUP BY stage;
