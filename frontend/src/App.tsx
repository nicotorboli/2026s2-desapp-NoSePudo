import React, { useState } from 'react';
import { Navbar } from './components/Navbar/Navbar';
import { PlayerListPage } from './pages/PlayerListPage/PlayerListPage';
import { PlayerDetailPage } from './pages/PlayerDetailPage/PlayerDetailPage';
import { RankingPage } from './pages/RankingPage/RankingPage';
import { AdminPage } from './pages/AdminPage/AdminPage';
import { useAdminSession } from './auth/adminSession';

type View = 'catalog' | 'detail' | 'ranking' | 'admin';

const App: React.FC = () => {
  const [view, setView] = useState<View>('catalog');
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null);
  const { isSuperuser } = useAdminSession();

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

  const handleNavigateAdmin = () => {
    setSelectedPlayerId(null);
    setView('admin');
  };

  return (
    <div className="app-container">
      <Navbar
        onNavigateHome={handleBackToCatalog}
        onNavigateRanking={handleNavigateRanking}
        onNavigateAdmin={handleNavigateAdmin}
        isAdmin={isSuperuser}
      />
      {view === 'catalog' && <PlayerListPage onSelectPlayer={handleSelectPlayer} />}
      {view === 'detail' && selectedPlayerId !== null && (
        <PlayerDetailPage playerId={selectedPlayerId} onBack={handleBackToCatalog} />
      )}
      {view === 'ranking' && <RankingPage onSelectPlayer={handleSelectPlayer} />}
      {view === 'admin' && <AdminPage />}
    </div>
  );
};

export default App;
