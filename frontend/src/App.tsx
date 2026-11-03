import React, { useState } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import { Navbar } from './components/Navbar/Navbar';
import { AuthModal } from './components/AuthModal/AuthModal';
import { PlayerListPage } from './pages/PlayerListPage/PlayerListPage';
import { PlayerDetailPage } from './pages/PlayerDetailPage/PlayerDetailPage';
import { PortfolioPage } from './pages/PortfolioPage/PortfolioPage';

const AppContent: React.FC = () => {
  const { isAuthenticated } = useAuth();
  const [currentView, setCurrentView] = useState<'catalog' | 'portfolio'>('catalog');
  const [selectedPlayerId, setSelectedPlayerId] = useState<number | null>(null);
  const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
  const [authModalTab, setAuthModalTab] = useState<'login' | 'register'>('login');

  const handleOpenAuth = (tab: 'login' | 'register' = 'login') => {
    setAuthModalTab(tab);
    setIsAuthModalOpen(true);
  };

  const handleSelectPlayer = (id: number) => {
    setSelectedPlayerId(id);
    setCurrentView('catalog');
    if (!isAuthenticated) {
      handleOpenAuth('login');
    }
  };

  const handleBackToCatalog = () => {
    setSelectedPlayerId(null);
    setCurrentView('catalog');
  };

  const handleNavigateView = (view: 'catalog' | 'portfolio') => {
    setCurrentView(view);
    if (view === 'catalog') {
      setSelectedPlayerId(null);
    }
  };

  return (
    <div className="app-container">
      <Navbar
        currentView={currentView}
        onNavigateHome={handleBackToCatalog}
        onNavigateView={handleNavigateView}
        onOpenAuth={() => handleOpenAuth('login')}
      />

      {currentView === 'portfolio' ? (
        <PortfolioPage
          onSelectPlayer={handleSelectPlayer}
          onBackToCatalog={handleBackToCatalog}
        />
      ) : selectedPlayerId === null ? (
        <PlayerListPage onSelectPlayer={handleSelectPlayer} />
      ) : (
        <PlayerDetailPage
          playerId={selectedPlayerId}
          onBack={handleBackToCatalog}
          onOpenAuth={handleOpenAuth}
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
