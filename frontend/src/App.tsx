import React, { useEffect, useState } from 'react';
import { getHealth, HealthResponseDTO } from './api/health';

const App: React.FC = () => {
  const [health, setHealth] = useState<HealthResponseDTO | null>(null);

  useEffect(() => {
    getHealth()
      .then(setHealth)
      .catch(() => setHealth({ status: 'offline', timestamp: '' }));
  }, []);

  return (
    <div className="app">
      <h1>NoSePudo</h1>
      <p>Status: {health ? health.status : 'loading...'}</p>
    </div>
  );
};

export default App;
