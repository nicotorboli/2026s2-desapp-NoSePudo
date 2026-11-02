import React, { useState } from 'react';
import { Navbar } from './components/Navbar/Navbar';
import { PlayerListPage } from './pages/PlayerListPage/PlayerListPage';
import { PlayerDetailPage } from './pages/PlayerDetailPage/PlayerDetailPage';
import { RankingPage } from './pages/RankingPage/RankingPage';

type View = 'catalog' | 'detail' | 'ranking';

const App: React.FC = () => {
  const [view, setView] = useState<View>('catalog');
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null);

  const handleSelectPlayer = (id: number) => {
    setSelectedPlayerId(id);
    setView('detail');
  };

  const handleBackToCatalog = () => {
    setSelectedPlayerId(null);
    setView('catalog');
  };

  const handleNavigateRanking = () => {
    setSelectedPlayerId(null);
    setView('ranking');
  };

  return (
    <div className="app-container">
      <Navbar onNavigateHome={handleBackToCatalog} onNavigateRanking={handleNavigateRanking} />
      {view === 'catalog' && <PlayerListPage onSelectPlayer={handleSelectPlayer} />}
      {view === 'detail' && selectedPlayerId !== null && (
        <PlayerDetailPage playerId={selectedPlayerId} onBack={handleBackToCatalog} />
      )}
      {view === 'ranking' && <RankingPage onSelectPlayer={handleSelectPlayer} />}
    </div>
  );
};

export default App;
