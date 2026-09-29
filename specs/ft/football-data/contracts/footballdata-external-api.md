# External Contract: football-data.org API v4 Integration

**Feature Branch**: `ft/football-data`  
**Date**: 2026-09-28  
**Component**: `internal/adapters/footballdata/`

## 1. Overview
The `football-data` adapter communicates with `https://api.football-data.org/v4` to retrieve official competition teams and squad members for the top five European leagues.

## 2. Authentication Contract
- **HTTP Header**: `X-Auth-Token: <API_KEY>`
- **Environment Variable**: `NSPFOOTBALLDATAAPIKEY` 
- If the token is empty, the adapter returns a configuration error during startup/sync invocation without crashing the application.

## 3. Endpoints & Operations

### 3.1 List Competition Teams
- **Method**: `GET`
- **Path**: `/v4/competitions/{competition_code}/teams`
- **Competition Codes**:
  - `PL`: Premier League (England)
  - `BL1`: Bundesliga (Germany)
  - `PD`: La Liga (Spain)
  - `SA`: Serie A (Italy)
  - `FL1`: Ligue 1 (France)
- **Response Schema (subset used)**:
```json
{
  "competition": {
    "id": 2021,
    "name": "Premier League",
    "code": "PL"
  },
  "season": {
    "id": 1564,
    "startDate": "2026-08-15",
    "endDate": "2027-05-25"
  },
  "teams": [
    {
      "id": 57,
      "name": "Arsenal FC",
      "shortName": "Arsenal",
      "tla": "ARS",
      "squad": [ ... ]
    }
  ]
}
```

### 3.2 Get Team Squad (Fallback if `teams[i].squad` is empty)
- **Method**: `GET`
- **Path**: `/v4/teams/{team_id}`
- **Response Schema (subset used)**:
```json
{
  "id": 57,
  "name": "Arsenal FC",
  "squad": [
    {
      "id": 7821,
      "name": "Bukayo Saka",
      "position": "Offence",
      "dateOfBirth": "2001-09-05",
      "nationality": "England",
      "shirtNumber": 7
    }
  ]
}
```

## 4. Rate Limiting Contract & HTTP 429
- Free tier limit: 10 requests / minute.
- Standard delay between requests: 6 seconds (configurable, 0ms in unit/mock tests).
- On HTTP 429:
  - Read `Retry-After` or `X-RequestCounter-Reset` header.
  - Wait specified backoff duration or abort if retry limit (3 retries) is exceeded.
  - Return distinct error `ErrRateLimitExceeded`.

## 5. Domain Mapping Rules
| External Field (`squad[]`) | Domain Model (`model.Player`) | Transformation / Fallback |
|---|---|---|
| `id` | `ExternalID` | Direct int64 conversion |
| `name` | `Name` | String trimmed; required |
| `position` | `Position` | Normalized: `"Goalkeeper"`, `"Defence"` → `"Defender"`, `"Midfield"` → `"Midfielder"`, `"Offence"` → `"Attacker"` |
| `dateOfBirth` | `DateOfBirth` | Parsed ISO date string (`YYYY-MM-DD`); nil if null |
| `nationality` | `Nationality` | String pointer; nil if null/empty |
| `shirtNumber` | `ShirtNumber` | Int pointer; nil if null or 0 |
| `team.name` | `ClubName` | Direct string |
| `competition.name` | `LeagueName` | Direct string |
| `competition.code` | `LeagueCode` | Direct string |
