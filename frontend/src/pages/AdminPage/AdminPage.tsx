import React from 'react';
import { useAdminSession } from '../../auth/adminSession';
import { AdminLogin } from '../../components/admin/AdminLogin/AdminLogin';
import { SyncSection } from '../../components/admin/SyncSection/SyncSection';
import { StrategySection } from '../../components/admin/StrategySection/StrategySection';
import { CoefficientSection } from '../../components/admin/CoefficientSection/CoefficientSection';
import { CatalogSection } from '../../components/admin/CatalogSection/CatalogSection';
import './AdminPage.css';

export const AdminPage: React.FC = () => {
  const { session, isSuperuser, login, logout } = useAdminSession();

  if (!session) {
    return <AdminLogin onLogin={login} />;
  }

  if (!isSuperuser) {
    return (
      <main className="admin-page admin-page--denied">
        <div className="admin-page__denied-card">
          <h1 className="admin-page__title">Acceso restringido</h1>
          <p className="admin-page__subtitle">
            La cuenta {session.email} no tiene privilegios de superusuario.
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
          <span className="admin-page__email">{session.email}</span>
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
        <a className="admin-page__nav-link" href="#admin-catalogo">
          Catálogo
        </a>
      </nav>

      <div className="admin-page__sections">
        <SyncSection />
        <StrategySection />
        <CoefficientSection />
        <CatalogSection />
      </div>
    </main>
  );
};
