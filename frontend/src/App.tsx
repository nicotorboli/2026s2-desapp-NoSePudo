import React, { useState } from 'react';
import { AuthProvider } from './context/AuthContext';
import { Navbar } from './components/Navbar/Navbar';
import { AuthModal } from './components/AuthModal/AuthModal';
import { PlayerListPage } from './pages/PlayerListPage/PlayerListPage';
import { PlayerDetailPage } from './pages/PlayerDetailPage/PlayerDetailPage';
import { RankingPage } from './pages/RankingPage/RankingPage';
import { AdminPage } from './pages/AdminPage/AdminPage';
import { PortfolioPage } from './pages/PortfolioPage/PortfolioPage';

type View = 'catalog' | 'detail' | 'ranking' | 'admin' | 'portfolio';

const AppContent: React.FC = () => {
  const [view, setView] = useState<View>('catalog');
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null);
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
  const [authModalTab, setAuthModalTab] = useState<'login' | 'register'>('login');

  const handleOpenAuth = (tab: 'login' | 'register' = 'login') => {
    setAuthModalTab(tab);
    setIsAuthModalOpen(true);
  };

  const handleSelectPlayer = (id: number) => {
    setSelectedPlayerId(id);
    setView('detail');
  };

  const handleBackToCatalog = () => {
    setSelectedPlayerId(null);
    setView('catalog');
  };

  const handleNavigate = (next: View) => {
    if (next !== 'detail') {
      setSelectedPlayerId(null);
    }
    setView(next);
  };

  return (
    <div className="app-container">
      <Navbar
        currentView={view}
        onNavigate={handleNavigate}
        onOpenAuth={() => handleOpenAuth('login')}
      />

      {view === 'catalog' && <PlayerListPage onSelectPlayer={handleSelectPlayer} />}
      {view === 'detail' && selectedPlayerId !== null && (
        <PlayerDetailPage
          playerId={selectedPlayerId}
          onBack={handleBackToCatalog}
          onOpenAuth={handleOpenAuth}
        />
      )}
      {view === 'ranking' && <RankingPage onSelectPlayer={handleSelectPlayer} />}
      {view === 'admin' && <AdminPage onOpenAuth={() => handleOpenAuth('login')} />}
      {view === 'portfolio' && (
        <PortfolioPage
          onSelectPlayer={handleSelectPlayer}
          onBackToCatalog={handleBackToCatalog}
        />
      )}

      <AuthModal
        isOpen={isAuthModalOpen}
        initialTab={authModalTab}
        onClose={() => setIsAuthModalOpen(false)}
      />
    </div>
  );
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
};

export default App;
