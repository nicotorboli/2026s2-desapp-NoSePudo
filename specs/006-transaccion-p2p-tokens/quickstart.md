# Quickstart – Validación de Transacción P2P de Tokens

**Feature**: `006-transaccion-p2p-tokens`
**Fecha**: 2026-11-03

---

## Prerequisitos

```bash
# Desde la raíz del repo
docker compose up -d          # Postgres en :5433

# Variables de entorno mínimas
export NSPPSQLDS="postgres://devuser:devpassword@localhost:5433/nsp_db?sslmode=disable"
export NSP_JWT_SECRET="dev-secret-de-al-menos-32-caracteres-ok"

cd backend
go run ./cmd/server           # API en :8080
```

Frontend (en otra terminal):
```bash
cd frontend
npm run dev                   # En :5173
```

---

## Escenario 1 – Flujo feliz completo

### 1.1 Registrar vendedor y comprador

```bash
# Vendedor
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"vendedor@test.com","password":"Vendedor123!"}' | jq .

# Comprador
curl -s -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"comprador@test.com","password":"Comprador123!"}' | jq .
```

### 1.2 Login y obtener tokens JWT

```bash
VENDEDOR_TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"vendedor@test.com","password":"Vendedor123!"}' | jq -r .accessToken)

COMPRADOR_TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"comprador@test.com","password":"Comprador123!"}' | jq -r .accessToken)
```

> **Nota**: El vendedor necesita tener tokens previamente comprados en el mercado abierto (`POST /orders/buy`) antes de poder crear una sesión P2P.

### 1.3 Crear sesión P2P

```bash
SESSION=$(curl -s -X POST http://localhost:8080/p2p/sessions \
  -H "Authorization: Bearer $VENDEDOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 5, "pricePerToken": 10000}')

echo $SESSION | jq .

SESSION_TOKEN=$(echo $SESSION | jq -r .token)
SESSION_URL=$(echo $SESSION | jq -r .sessionUrl)
```

**Resultado esperado**:
- HTTP 201
- `status: "PENDING"`
- `expiresAt` ≈ `createdAt + 5 min`
- `sessionUrl` contiene el UUID del token
- Los 5 tokens del vendedor aparecen en `reservedQuantity` al consultar su portfolio

### 1.4 Comprador accede y acepta

```bash
# Ver detalles de la sesión
curl -s -X GET "http://localhost:8080/p2p/sessions/$SESSION_TOKEN" \
  -H "Authorization: Bearer $COMPRADOR_TOKEN" | jq .
```

**Resultado esperado**:
- Muestra `seller`, `player`, `quantity`, `pricePerToken`, `totalAmount`, `expiresAt`
- `buyer` es null

```bash
# Aceptar
curl -s -X POST "http://localhost:8080/p2p/sessions/$SESSION_TOKEN/accept" \
  -H "Authorization: Bearer $COMPRADOR_TOKEN" | jq .
```

**Resultado esperado**:
- HTTP 200
- Respuesta con `tradeId`, `buyerId`, `sellerId`, `quantity: 5`, `totalAmount: 50000`
- La sesión pasa a `COMPLETED`
- El portfolio del comprador refleja +5 tokens del jugador
- El portfolio del vendedor refleja -5 tokens y +50000 créditos

---

## Escenario 2 – Expiración automática

```bash
# Crear sesión
SESSION_TOKEN=$(curl -s -X POST http://localhost:8080/p2p/sessions \
  -H "Authorization: Bearer $VENDEDOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 3, "pricePerToken": 10000}' | jq -r .token)

# Esperar 5+ minutos (el scheduler expira la sesión en hasta 30 s extra)
sleep 330

# Intentar aceptar → debe fallar
curl -s -X POST "http://localhost:8080/p2p/sessions/$SESSION_TOKEN/accept" \
  -H "Authorization: Bearer $COMPRADOR_TOKEN" | jq .
```

**Resultado esperado**: HTTP 410, `error: "la sesión P2P ya no está activa"`. Los tokens del vendedor están liberados (`reservedQuantity` volvió a 0).

---

## Escenario 3 – Cancelación por el vendedor

```bash
SESSION_TOKEN=$(curl -s -X POST http://localhost:8080/p2p/sessions \
  -H "Authorization: Bearer $VENDEDOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 2, "pricePerToken": 10000}' | jq -r .token)

curl -s -X POST "http://localhost:8080/p2p/sessions/$SESSION_TOKEN/cancel" \
  -H "Authorization: Bearer $VENDEDOR_TOKEN"
```

**Resultado esperado**: HTTP 204. Sesión pasa a `CANCELLED`. Tokens liberados inmediatamente.

---

## Escenario 4 – Rechazo de auto-trade

```bash
curl -s -X POST "http://localhost:8080/p2p/sessions/$SESSION_TOKEN/accept" \
  -H "Authorization: Bearer $VENDEDOR_TOKEN" | jq .
```

**Resultado esperado**: HTTP 409, `error: "el vendedor no puede aceptar su propia oferta"`.

---

## Escenario 5 – Fondos insuficientes del comprador

Crear una sesión por un monto mayor al saldo disponible del comprador (comprador registrado con 1000 créditos, sesión con `pricePerToken * quantity > 1000 créditos`).

```bash
curl -s -X POST "http://localhost:8080/p2p/sessions/$SESSION_TOKEN/accept" \
  -H "Authorization: Bearer $COMPRADOR_TOKEN" | jq .
```

**Resultado esperado**: HTTP 422, `error: "el saldo disponible no alcanza"`. La sesión permanece en `PENDING`.

---

## Validación frontend

1. Abrir `http://localhost:5173` como vendedor → navegar a la sección P2P → crear oferta.
2. Copiar el link generado.
3. Abrir el link en navegador en modo incógnito → iniciar sesión como comprador.
4. Verificar que la vista muestra ambos usuarios, jugador, cantidad, precio total y countdown.
5. Pulsar "Aceptar" → verificar redirección/mensaje de éxito y actualización de portfolios.
6. Repetir con link expirado → verificar que el botón está deshabilitado y aparece el mensaje de sesión vencida.
