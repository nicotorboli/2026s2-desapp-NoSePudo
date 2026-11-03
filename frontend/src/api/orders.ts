import client from './client';

export type TimeInForce = 'GTC' | 'IOC' | 'FOK';
export type OrderSide = 'BUY' | 'SELL';
export type OrderStatus = 'PENDING' | 'PARTIALLY_FILLED' | 'FILLED' | 'CANCELLED';

export interface PlaceOrderRequestDTO {
  playerId: number;
  quantity: number;
  price?: string | number;
  timeInForce?: TimeInForce;
}

export interface ExecutionDTO {
  tradeId: number;
  quantity: number;
  price: string | number;
  executedAt: string;
}

export interface OrderResponseDTO {
  id: number;
  side: OrderSide;
  playerId: number;
  price: string | number;
  quantity: number;
  executedQuantity: number;
  remainingQuantity: number;
  timeInForce: TimeInForce;
  status: OrderStatus;
  createdAt: string;
  trades?: ExecutionDTO[];
}

export interface OpenOrderDTO {
  id: number;
  side: OrderSide;
  playerId: number;
  userId?: number;
  price: string | number;
  quantity: number;
  executedQuantity: number;
  remainingQuantity: number;
  timeInForce: TimeInForce;
  status: OrderStatus;
  createdAt: string;
}

export interface CancelOrderResponseDTO {
  id: number;
  status: OrderStatus;
  releasedCredits?: string | number;
  releasedTokens?: number;
}

const formatOrderPayload = (order: PlaceOrderRequestDTO) => {
  const payload: Record<string, unknown> = {
    playerId: order.playerId,
    quantity: order.quantity,
  };
  if (order.price !== undefined && order.price !== null) {
    payload.price = typeof order.price === 'number' ? order.price.toFixed(4) : order.price;
  }
  if (order.timeInForce) {
    payload.timeInForce = order.timeInForce;
  }
  return payload;
};

const post = async <T, B>(url: string, body: B): Promise<T> => {
  const response = await client.post<T>(url, body);
  return response.data;
};

const get = async <T>(url: string, params?: Record<string, unknown>): Promise<T> => {
  const response = await client.get<T>(url, { params });
  return response.data;
};

const del = async <T>(url: string): Promise<T> => {
  const response = await client.delete<T>(url);
  return response.data;
};

export const placeBuyOrder = async (order: PlaceOrderRequestDTO): Promise<OrderResponseDTO> => {
  return post<OrderResponseDTO, Record<string, unknown>>('/orders/buy', formatOrderPayload(order));
};

export const placeSellOrder = async (order: PlaceOrderRequestDTO): Promise<OrderResponseDTO> => {
  return post<OrderResponseDTO, Record<string, unknown>>('/orders/sell', formatOrderPayload(order));
};

export const getOpenOrders = async (playerId?: number, market?: boolean): Promise<OpenOrderDTO[]> => {
  const params: Record<string, unknown> = {};
  if (playerId) params.playerId = playerId;
  if (market) params.market = true;
  const res = await get<{ items?: OpenOrderDTO[] } | OpenOrderDTO[]>(
    '/orders',
    Object.keys(params).length > 0 ? params : undefined
  );
  if (Array.isArray(res)) {
    return res;
  }
  return res.items || [];
};

export const cancelOrder = async (orderId: number): Promise<CancelOrderResponseDTO> => {
  return del<CancelOrderResponseDTO>(`/orders/${orderId}`);
};
