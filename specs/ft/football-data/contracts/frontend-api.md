# Frontend Interface Contract: Player Catalog & Details

**Feature Branch**: `ft/football-data`  
**Date**: 2026-09-28  
**Component**: `frontend/src/`

## 1. API Abstraction Module (`frontend/src/api/players.ts`)

In accordance with Constitution Principle VI:
- Axios is imported **only** within API modules (e.g. `src/api/`).
- Endpoint callers use exported functions.
- TypeScript interfaces mirror backend DTOs.

### 1.1 TypeScript DTOs

```typescript
// Matches backend PlayerListItemResponse
export interface PlayerListItemDTO {
  id: number;
  name: string;
  club: string;
  league: string;
  position: string;
}

// Matches backend PaginatedResponse<PlayerListItemResponse>
export interface PaginatedPlayersDTO {
  items: PlayerListItemDTO[];
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

// Matches backend PlayerDetailResponse
export interface PlayerDetailDTO {
  id: number;
  externalId: number;
  name: string;
  club: string;
  league: string;
  leagueCode: string;
  position: string;
  dateOfBirth: string | null;
  nationality: string | null;
  shirtNumber: number | null;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

// Query parameters for player filtering
export interface PlayerFilterParams {
  page?: number;
  limit?: number;
  league?: string;
  club?: string;
  position?: string;
  search?: string;
  includeInactive?: boolean;
}
```

### 1.2 Exported API Functions

```typescript
export const getPlayers = async (params?: PlayerFilterParams): Promise<PaginatedPlayersDTO> => { ... };
export const getPlayerById = async (id: number): Promise<PlayerDetailDTO> => { ... };
```

---

## 2. Page & Component Contracts

### 2.1 `PlayerListPage` (`frontend/src/pages/PlayerListPage/`)
- **Route / View**: Catalog default view (`/` or `/players`).
- **Displays**:
  - Filter bar: league dropdown (All, Premier League, Bundesliga, La Liga, Serie A, Ligue 1), club name text filter, position dropdown (All, Goalkeeper, Defender, Midfielder, Attacker), and search input.
  - Active/historical toggle (optional view for inactive players).
  - Player table/card list: renders only **name**, **current team (club)**, **league**, and **position** as requested.
  - Clicking on a player row or card triggers navigation to `PlayerDetailPage` with that player's `id`.
  - Pagination bar: current page, total pages, total count, Previous / Next buttons.
  - Empty state indicator when no players match filters.
  - Loading skeleton / spinner during API fetches.
- **BEM CSS File**: `PlayerListPage.css`
  - Blocks: `.player-list-page`
  - Elements: `.player-list-page__header`, `.player-list-page__title`, `.player-list-page__content`, `.player-list-page__empty`

### 2.2 `PlayerDetailPage` (`frontend/src/pages/PlayerDetailPage/`)
- **Route / View**: Single player view (`/players/:id`).
- **Displays**:
  - Back button navigating back to catalog list.
  - Player header: full name, active status badge (`Active` vs `Inactive/Departed`).
  - Full attribute grid/cards:
    - Club / Current Team
    - League & Competition Code
    - Primary Playing Position
    - Date of Birth (formatted, e.g. "September 5, 2001", or "—" if unassigned)
    - Nationality (with flag/label or "Unknown")
    - Squad Shirt Number (e.g. "#7" or "Unassigned")
    - External Provider ID
    - Last Synchronized / Updated Timestamp
  - Error state with friendly notification if player is not found.
- **BEM CSS File**: `PlayerDetailPage.css`
  - Blocks: `.player-detail-page`, `.player-card`
  - Elements: `.player-card__header`, `.player-card__badge--active`, `.player-card__badge--inactive`, `.player-card__field`, `.player-card__label`, `.player-card__value`

### 2.3 `PlayerFilter` Component (`frontend/src/components/PlayerFilter/`)
- **BEM Classes**: `.player-filter`, `.player-filter__group`, `.player-filter__input`, `.player-filter__select`, `.player-filter__button`

### 2.4 `Pagination` Component (`frontend/src/components/Pagination/`)
- **BEM Classes**: `.pagination`, `.pagination__button`, `.pagination__button--disabled`, `.pagination__info`
