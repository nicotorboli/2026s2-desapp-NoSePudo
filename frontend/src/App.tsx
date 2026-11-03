import React, { useState } from 'react';
import { AuthProvider } from './context/AuthContext';
import { Navbar } from './components/Navbar/Navbar';
import { AuthModal } from './components/AuthModal/AuthModal';
import { PlayerListPage } from './pages/PlayerListPage/PlayerListPage';
import { PlayerDetailPage } from './pages/PlayerDetailPage/PlayerDetailPage';
import { RankingPage } from './pages/RankingPage/RankingPage';
import { AdminPage } from './pages/AdminPage/AdminPage';
import { PortfolioPage } from './pages/PortfolioPage/PortfolioPage';
import { P2PCreatePage } from './pages/P2PCreatePage/P2PCreatePage';
import { P2PSessionPage } from './pages/P2PSessionPage/P2PSessionPage';

type View = 'catalog' | 'detail' | 'ranking' | 'admin' | 'portfolio' | 'p2p-create' | 'p2p-session';

const AppContent: React.FC = () => {
  const [view, setView] = useState<View>('catalog');
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null);
  const [p2pToken, setP2PToken] = useState<string | null>(null);
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
  const [authModalTab, setAuthModalTab] = useState<'login' | 'register'>('login');

  React.useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get('token');
    if (token) {
      setP2PToken(token);
      setView('p2p-session');
    }
  }, []);

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
      {view === 'p2p-create' && (
        <P2PCreatePage
          onBack={handleBackToCatalog}
          onOpenSession={(token) => {
            setP2PToken(token);
            setView('p2p-session');
          }}
          onOpenAuth={() => handleOpenAuth('login')}
        />
      )}
      {view === 'p2p-session' && p2pToken && (
        <P2PSessionPage
          token={p2pToken}
          onBack={handleBackToCatalog}
          onOpenAuth={() => handleOpenAuth('login')}
          onGoToPortfolio={() => handleNavigate('portfolio')}
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
