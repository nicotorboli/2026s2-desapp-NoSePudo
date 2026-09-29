const positionTranslations: Record<string, string> = {
  Goalkeeper: 'Arquero',
  Defender: 'Defensor',
  Midfielder: 'Mediocampista',
  Attacker: 'Delantero',
};

export const translatePosition = (position: string): string => {
  const normalized = position.trim();
  const key = Object.keys(positionTranslations).find(
    (k) => k.toLowerCase() === normalized.toLowerCase()
  );
  return key ? positionTranslations[key] : position;
};
