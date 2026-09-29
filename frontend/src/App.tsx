import React, { useState } from 'react';
import { Navbar } from './components/Navbar/Navbar';
import { PlayerListPage } from './pages/PlayerListPage/PlayerListPage';
import { PlayerDetailPage } from './pages/PlayerDetailPage/PlayerDetailPage';

const App: React.FC = () => {
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null);

  const handleSelectPlayer = (id: number) => {
    setSelectedPlayerId(id);
  };

  const handleBackToCatalog = () => {
    setSelectedPlayerId(null);
  };

  return (
    <div className="app-container">
      <Navbar onNavigateHome={handleBackToCatalog} />
      {selectedPlayerId === null ? (
        <PlayerListPage onSelectPlayer={handleSelectPlayer} />
      ) : (
        <PlayerDetailPage playerId={selectedPlayerId} onBack={handleBackToCatalog} />
      )}
    </div>
  );
};

export default App;
