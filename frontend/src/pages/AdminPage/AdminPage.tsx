import React from 'react';
import { useAuth } from '../../context/AuthContext';
import { SyncSection } from '../../components/admin/SyncSection/SyncSection';
import { StrategySection } from '../../components/admin/StrategySection/StrategySection';
import { CoefficientSection } from '../../components/admin/CoefficientSection/CoefficientSection';
import './AdminPage.css';

interface AdminPageProps {
  onOpenAuth?: () => void;
}

export const AdminPage: React.FC<AdminPageProps> = ({ onOpenAuth }) => {
  const { user, isAuthenticated, logout } = useAuth();
  const isSuperuser = user?.privilege === 'superuser';

  if (!isAuthenticated) {
    return (
      <main className="admin-page admin-page--denied">
        <div className="admin-page__denied-card">
          <h1 className="admin-page__title">Acceso restringido</h1>
          <p className="admin-page__subtitle">
            Iniciá sesión con una cuenta de superusuario para administrar el catálogo.
          </p>
          {onOpenAuth && (
            <button type="button" className="admin-page__logout" onClick={onOpenAuth}>
              Iniciar Sesión
            </button>
          )}
        </div>
      </main>
    );
  }

  if (!isSuperuser) {
    return (
      <main className="admin-page admin-page--denied">
        <div className="admin-page__denied-card">
          <h1 className="admin-page__title">Acceso restringido</h1>
          <p className="admin-page__subtitle">
            La cuenta {user?.email} no tiene privilegios de superusuario.
          </p>
          <button
            type="button"
            className="admin-page__logout"
            onClick={() => void logout()}
          >
            Cerrar sesión
          </button>
        </div>
      </main>
    );
  }

  return (
    <main className="admin-page">
      <header className="admin-page__header">
        <div>
          <h1 className="admin-page__title">Administración</h1>
          <p className="admin-page__subtitle">
            Configuración del catálogo, scoring y sincronizaciones.
          </p>
        </div>
        <div className="admin-page__session">
          <span className="admin-page__email">{user?.email}</span>
          <button
            type="button"
            className="admin-page__logout"
            onClick={() => void logout()}
          >
            Cerrar sesión
          </button>
        </div>
      </header>

      <nav className="admin-page__nav" aria-label="Secciones de administración">
        <a className="admin-page__nav-link" href="#admin-sync">
          Sincronizaciones
        </a>
        <a className="admin-page__nav-link" href="#admin-estrategias">
          Estrategias
        </a>
        <a className="admin-page__nav-link" href="#admin-coeficientes">
          Coeficientes
        </a>
      </nav>

      <div className="admin-page__sections">
        <SyncSection />
        <StrategySection />
        <CoefficientSection />
      </div>
    </main>
  );
};
