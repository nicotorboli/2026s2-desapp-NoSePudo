import React from 'react';
import './Pagination.css';

interface PaginationProps {
  page: number;
  totalPages: number;
  total: number;
  onPageChange: (page: number) => void;
}

export const Pagination: React.FC<PaginationProps> = ({
  page,
  totalPages,
  total,
  onPageChange,
}) => {
  if (totalPages <= 1 && total <= 0) {
    return null;
  }

  const isFirstPage = page <= 1;
  const isLastPage = page >= totalPages;

  return (
    <nav className="pagination" aria-label="Navegación de paginación">
      <button
        type="button"
        className={`pagination__button ${isFirstPage ? 'pagination__button--disabled' : ''}`}
        disabled={isFirstPage}
        onClick={() => onPageChange(page - 1)}
        aria-label="Página anterior"
      >
        ← Anterior
      </button>

      <span className="pagination__info">
        Página <strong>{page}</strong> de <strong>{totalPages || 1}</strong> ({total} jugadores en total)
      </span>

      <button
        type="button"
        className={`pagination__button ${isLastPage ? 'pagination__button--disabled' : ''}`}
        disabled={isLastPage}
        onClick={() => onPageChange(page + 1)}
        aria-label="Página siguiente"
      >
        Siguiente →
      </button>
    </nav>
  );
};
