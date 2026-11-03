const positionTranslations: Record<string, string> = {
  Goalkeeper: 'Arquero',
  Defender: 'Defensor',
  Midfielder: 'Mediocampista',
  Attacker: 'Delantero',
};

const statTranslations: Record<string, string> = {
  goals: 'Goles',
  assists: 'Asistencias',
  shots: 'Remates',
  key_passes: 'Pases clave',
  dribbles: 'Regates',
  tackles: 'Entradas',
  interceptions: 'Intercepciones',
  fouls: 'Faltas',
  clears: 'Despejes',
  blocks: 'Bloqueos',
  was_dribbled: 'Gambeteado',
};

export const translatePosition = (position: string): string => {
  const normalized = position.trim();
  const key = Object.keys(positionTranslations).find(
    (k) => k.toLowerCase() === normalized.toLowerCase()
  );
  return key ? positionTranslations[key] : position;
};

export const translateStat = (stat: string): string => {
  const normalized = stat.trim().toLowerCase();
  return statTranslations[normalized] ?? stat;
};
